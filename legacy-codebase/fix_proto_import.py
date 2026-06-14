import os
import re

filepath = "new-backend/monolith/internal/modules/meal/service/payment_client.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('"github.com/baaaki/mydreamcampus/meal-service/proto"', '"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/proto"')

with open(filepath, 'w') as f:
    f.write(content)
print("Done fix proto import")
