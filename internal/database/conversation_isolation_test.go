package database

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// 回归测试:对话严格按 owner 隔离,即使 scope=all(管理员)也只看到自己的对话。
// 对应线上 bug:所有用户(含 admin)都能看到全部对话记录。
func TestConversationStrictIsolationForScopeAll(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "iso.db"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// 引导系统角色(admin/operator/viewer/auditor),否则 user_roles 外键失败
	if err := db.BootstrapRBAC("hash", map[string]string{"auth:self": "self"}); err != nil {
		t.Fatal(err)
	}

	// 两个用户都授予 admin 角色(scope=all),模拟线上全部用户皆管理员的情况
	alice, err := db.CreateRBACUser("iso-alice", "Alice", "hash", true, []string{RBACSystemRoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := db.CreateRBACUser("iso-bob", "Bob", "hash", true, []string{RBACSystemRoleAdmin})
	if err != nil {
		t.Fatal(err)
	}

	convA, err := db.CreateConversation("A 的对话", ConversationCreateMeta{})
	if err != nil {
		t.Fatal(err)
	}
	convB, err := db.CreateConversation("B 的对话", ConversationCreateMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetResourceOwner("conversation", convA.ID, alice.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.SetResourceOwner("conversation", convB.ID, bob.ID); err != nil {
		t.Fatal(err)
	}

	// 1) 列表:scope=all 时仍严格按 owner 过滤
	rowsA, err := db.ListConversationsForAccess(50, 0, "", "", "", alice.ID, RBACScopeAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsA) != 1 || rowsA[0].ID != convA.ID {
		t.Fatalf("alice conversations = %#v, want only %s", rowsA, convA.ID)
	}
	rowsB, err := db.ListConversationsForAccess(50, 0, "", "", "", bob.ID, RBACScopeAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsB) != 1 || rowsB[0].ID != convB.ID {
		t.Fatalf("bob conversations = %#v, want only %s", rowsB, convB.ID)
	}

	// 2) 单条:UserCanAccessResource(scope=all) 也必须严格(handler 与准备流程都走它)
	if db.UserCanAccessResource(alice.ID, RBACScopeAll, "conversation", convB.ID) {
		t.Fatal("alice must NOT access bob's conversation even with scope=all")
	}
	if !db.UserCanAccessResource(alice.ID, RBACScopeAll, "conversation", convA.ID) {
		t.Fatal("alice should access her own conversation")
	}
	if !db.UserCanAccessConversationStrict(bob.ID, convB.ID) {
		t.Fatal("bob should access his own conversation")
	}
	if db.UserCanAccessConversationStrict(bob.ID, convA.ID) {
		t.Fatal("bob must NOT access alice's conversation")
	}

	// 3) 消息级:按消息反查对话,同样严格隔离
	msgA, err := db.AddMessage(convA.ID, "user", "alice 的消息", nil)
	if err != nil {
		t.Fatal(err)
	}
	if db.UserCanAccessMessageStrict(bob.ID, msgA.ID) {
		t.Fatal("bob must NOT access alice's message")
	}
	if !db.UserCanAccessMessageStrict(alice.ID, msgA.ID) {
		t.Fatal("alice should access her own message")
	}
	// 兼容旧接口(scope 参数版本)同样严格
	if db.UserCanAccessMessage(bob.ID, RBACScopeAll, msgA.ID) {
		t.Fatal("old UserCanAccessMessage must be strict too")
	}
}

// EnsureAllUsersAdmin 幂等地让每个用户都带上 admin 角色(scope=all),
// 与对话隔离配合:功能全开,但数据按 owner 隔离。
func TestEnsureAllUsersAdmin(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "migrate.db"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.BootstrapRBAC("hash", map[string]string{"auth:self": "self"}); err != nil {
		t.Fatal(err)
	}
	operator, err := db.CreateRBACUser("old-operator", "Old", "hash", true, []string{RBACSystemRoleOperator})
	if err != nil {
		t.Fatal(err)
	}
	norole, err := db.CreateRBACUser("no-role", "NoRole", "hash", true, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.EnsureAllUsersAdmin(); err != nil {
		t.Fatal(err)
	}

	for name, id := range map[string]string{"old-operator": operator.ID, "no-role": norole.ID} {
		access, err := db.ResolveRBACAccess(id)
		if err != nil {
			t.Fatalf("resolve %s: %v", name, err)
		}
		if access.Scope != RBACScopeAll {
			t.Fatalf("%s scope = %q, want all after migration", name, access.Scope)
		}
	}

	// 幂等:再次执行不报错
	if err := db.EnsureAllUsersAdmin(); err != nil {
		t.Fatal(err)
	}
}
