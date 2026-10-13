# Antigravity Agent Guidelines for MyDreamCampus

## File Writing & IDE Bug Workaround (CRITICAL)
- **DO NOT use the UI-staged `write_to_file` tool**: In this VS Code environment, `write_to_file` triggers a transient approval modal that vanishes immediately, causing the editor buffer to discard unsaved content and leaving empty 0-byte files on disk.
- **ALWAYS write and modify files directly via shell commands (`run_command`)**:
  - For new files: Use standard bash redirection (`cat << 'EOF' > filepath ... EOF`).
  - For modifying files: Use python scripts or sed/awk/direct shell rewrites.
- **MANDATORY verification**: Immediately after writing or modifying any file, verify its physical size on disk with `ls -lh <filepath>` to guarantee it is NOT 0 bytes, and run relevant compiler/tests (`go test`, `npm test`) to ensure integrity.
