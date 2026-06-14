import os
import re

directory = "new-backend/monolith/internal/modules/meal"

replacements = {
    r'\bdb\.MealMealTimeEnum\b': 'db.MealTimeEnum',
    r'\bdb\.MealMenuTypeEnum\b': 'db.MenuTypeEnum',
    r'\bdb\.MealReservationStatusEnum\b': 'db.ReservationStatusEnum',
    r'\bdb\.MealOutboxStatusEnum\b': 'db.OutboxStatusEnum',
}

for root, dirs, files in os.walk(directory):
    for filename in files:
        if filename.endswith(".go") and "db/models.go" not in os.path.join(root, filename) and "db/queries.sql.go" not in os.path.join(root, filename):
            filepath = os.path.join(root, filename)
            with open(filepath, 'r') as f:
                content = f.read()
            
            for pattern, replacement in replacements.items():
                # We want to replace db.MealMealTimeEnum but NOT db.MealMealTimeEnumLunch
                # We can use negative lookahead
                content = re.sub(pattern + r'(?!Lunch|Dinner|Normal|Vegan|Pending|Confirmed|Cancelled|Expired|Published|Failed)', replacement, content)
                
            with open(filepath, 'w') as f:
                f.write(content)

print("Done Enum fix")
