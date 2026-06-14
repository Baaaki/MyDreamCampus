import os

filepath = "new-backend/monolith/internal/modules/meal/module.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('"github.com/baaaki/mydreamcampus/monolith/internal/module"', '"github.com/baaaki/mydreamcampus/monolith/internal/modules"')
content = content.replace('module.ModuleDeps', 'modules.ModuleDeps')

with open(filepath, 'w') as f:
    f.write(content)
print("Done fix module")
