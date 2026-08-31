---
name: ctf-solver
description: Autonomous CTF challenge-solving methodology — triage the challenge, run the right tools, extract the flag, and verify it with submit_flag.
whenToUse: Whenever asked to solve a capture-the-flag (CTF) challenge.
---

# CTF Challenge Solver

You are solving a capture-the-flag (CTF) challenge. Your only goal is to find the exact flag and verify it.

## Where the challenge lives

Each challenge is one directory under `challenges/`, e.g. `challenges/base64-101/`. Inside a challenge directory you will usually find:

- `README.md` — the challenge description and any hints.
- `challenge.txt` / `challenge.png` / `capture.pcap` / a binary / other attachment files — the material to analyze.

The flag format is usually `picoCTF{...}` or `CTF{...}`, but trust the actual description.

## Tools available on this machine

Native (always available): `file`, `xxd`, `python`, and the usual coreutils.

Python shims (on PATH, or call directly as `python tools/<name>.py`):
- `strings` — extract printable strings (`python tools/strings.py`)
- `binwalk` — scan a file for embedded signatures + PNG trailing data (`python tools/binwalk.py`)
- `zsteg` — extract LSB bit planes from an image and grep for flags (`python tools/zsteg.py`)
- `exiftool` — read image metadata (`python tools/exiftool.py`)

Python libraries already installed (use them in small scripts via `bash`):
- `PIL`/`pillow` — image pixel manipulation, LSB/stego extraction, metadata.
- `scapy` — pcap parsing (`rdpcap`, iterate packets, dump payloads).
- `Crypto`/`pycryptodome` + `cryptography` — AES/RSA/XOR, TLS decryption.

NOT installed (do NOT call these — they will error): `steghide`, `tshark`, `tcpdump`, `objdump`, `readelf`, `checksec`, `volatility`. Use the Python equivalents above instead.

## Shared findings (message bus)

Multiple solvers run in parallel during a competition. Cooperate through the shared findings bus:

- `read_findings(challenge="<name>")` — check what other solvers already learned before you start; call `read_findings()` with no argument to read every challenge's findings.
- `share_finding(challenge="<name>", note="...")` — once you confirm something concrete (an encoding, the flag format, a port), share it so other solvers don't re-derive it.

## Method

1. **Read the description first**: use `read_file` on `challenges/<name>/README.md`.
2. **Triage the category** from the description and file types (use `file <name>` on any attachment to see its true type):
   - Encoded text / numbers → decode: base64, hex, rot13, caesar, xor, url-encoding.
   - Images (PNG/JPG/GIF) → try `zsteg` (LSB), `exiftool` (metadata), `strings`, `binwalk` (embedded/trailing data). For pixel-level work use a Python script with `PIL`.
   - Network capture (`.pcap`) → `strings` for plaintext, or a Python script with `scapy` to parse packets and dump payloads.
   - Binaries (ELF/PE) → `file` to identify, then `strings` to extract embedded strings; write small Python scripts for any byte-level static analysis.
   - Web → `curl` the target, check `robots.txt`, page source, headers, cookies.
   - Forensics → `strings`, `binwalk`, plus Python for image/pcap forensics.
3. **Run tools via `bash`**. Prefer small one-off Python scripts for decoding, bit extraction, packet parsing, and math — do not hand-decode. Remember: for a binary/PNG/pcap, `read_file` may return garbage — use `file`, `strings`, `xxd`, and Python instead.
4. **Extract a candidate flag.**
5. **Verify it** with `submit_flag(challenge="<name>", flag="<candidate>")`:
   - `CORRECT` → you are done, report the flag.
   - `INCORRECT` → do NOT resubmit the same value. Try a different approach.
   - Submissions are rate-limited after a wrong answer (and some challenges cap attempts) — treat `submit_flag` as a scarce resource, not a way to test guesses.

## Rules

- The flag is NEVER stored inside the challenge directory — do not look for a `flag.txt` or any local answer file. You must actually solve the challenge and derive the flag from the material (or the remote service).
- Ignore placeholder flags such as `picoCTF{flag}` or `picoCTF{placeholder}` — they are never real.
- Derive the flag from the material; do not guess repeatedly. Verify a candidate locally first (e.g. re-encode a decoded flag to confirm it round-trips, check the flag format) and submit once, when confident.
- Cover maximum surface area: hidden files, metadata, multiple encodings, LSB bit planes, embedded files, trailing data, and common CTF tricks.
- Keep using tools until you have the flag or you have exhausted every angle.
