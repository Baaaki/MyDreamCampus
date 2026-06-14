import os
import re

go_dir = "new-backend/monolith/internal/modules/attendance"

replacements = {
    r'"github\.com/baaaki/mydreamcampus/attendance-service/internal/([^"]+)"': r'"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/\1"',
    r'"github\.com/baaaki/mydreamcampus/shared/([^"]+)"': r'"github.com/baaaki/mydreamcampus/monolith/internal/platform/\1"',
    r'"github\.com/baaaki/mydreamcampus/attendance-service/config"': r'"github.com/baaaki/mydreamcampus/monolith/config"',
}

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    new_content = content
    for pattern, repl in replacements.items():
        new_content = re.sub(pattern, repl, new_content)

    if new_content != content:
        with open(filepath, 'w') as f:
            f.write(new_content)
        print(f"Updated {filepath}")

for root, _, files in os.walk(go_dir):
    for file in files:
        if file.endswith('.go'):
            process_file(os.path.join(root, file))
