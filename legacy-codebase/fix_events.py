import os
import re

go_dir = "new-backend/monolith/internal/modules/attendance"

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Fix monolith/internal/platform/events -> shared/events
    content = content.replace('"github.com/baaaki/mydreamcampus/monolith/internal/platform/events"', '"github.com/baaaki/mydreamcampus/shared/events"')
    
    # Temporarily comment out monolith/internal/platform/client if it exists
    # content = content.replace('"github.com/baaaki/mydreamcampus/monolith/internal/platform/client"', '// "github.com/baaaki/mydreamcampus/monolith/internal/platform/client"')

    with open(filepath, 'w') as f:
        f.write(content)

for root, _, files in os.walk(go_dir):
    for file in files:
        if file.endswith('.go'):
            process_file(os.path.join(root, file))
