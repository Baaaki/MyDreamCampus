import os
import re

directory = "new-backend/monolith/internal/modules/meal"

replacements = {
    # Imports
    r'"github\.com/baaaki/mydreamcampus/meal-service/internal': '"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal',
    r'"github\.com/baaaki/mydreamcampus/meal-service/config"': '"github.com/baaaki/mydreamcampus/monolith/config"',
    r'"github\.com/baaaki/mydreamcampus/shared/': '"github.com/baaaki/mydreamcampus/monolith/internal/platform/',
    
    # Types
    r'\bStudentsCache\b': 'StudentView',
    r'\bstudents_cache\b': 'students_view', # table name references in error msgs etc
    
    # Enums - sqlc added Meal prefix to values but not types
    # Actually sqlc generates MealTimeEnum, OutboxStatusEnum etc.
    # The constants were `db.MealTimeEnumLunch` etc. We need `db.MealMealTimeEnumLunch`
    r'\bdb\.MealTimeEnum': 'db.MealMealTimeEnum',
    r'\bdb\.MenuTypeEnum': 'db.MealMenuTypeEnum',
    r'\bdb\.ReservationStatusEnum': 'db.MealReservationStatusEnum',
    r'\bdb\.OutboxStatusEnum': 'db.MealOutboxStatusEnum',
    
    # But wait, we don't want to replace the type `db.MealTimeEnum` with `db.MealMealTimeEnum`
    # Let's fix only the values:
    r'\bdb\.MealTimeEnumLunch\b': 'db.MealMealTimeEnumLunch',
    r'\bdb\.MealTimeEnumDinner\b': 'db.MealMealTimeEnumDinner',
    r'\bdb\.MenuTypeEnumNormal\b': 'db.MealMenuTypeEnumNormal',
    r'\bdb\.MenuTypeEnumVegan\b': 'db.MealMenuTypeEnumVegan',
    r'\bdb\.ReservationStatusEnumPending\b': 'db.MealReservationStatusEnumPending',
    r'\bdb\.ReservationStatusEnumConfirmed\b': 'db.MealReservationStatusEnumConfirmed',
    r'\bdb\.ReservationStatusEnumCancelled\b': 'db.MealReservationStatusEnumCancelled',
    r'\bdb\.ReservationStatusEnumExpired\b': 'db.MealReservationStatusEnumExpired',
    r'\bdb\.OutboxStatusEnumPending\b': 'db.MealOutboxStatusEnumPending',
    r'\bdb\.OutboxStatusEnumPublished\b': 'db.MealOutboxStatusEnumPublished',
    r'\bdb\.OutboxStatusEnumFailed\b': 'db.MealOutboxStatusEnumFailed',
}

for root, dirs, files in os.walk(directory):
    for filename in files:
        if filename.endswith(".go") and "db/models.go" not in os.path.join(root, filename) and "db/queries.sql.go" not in os.path.join(root, filename):
            filepath = os.path.join(root, filename)
            with open(filepath, 'r') as f:
                content = f.read()
            
            for pattern, replacement in replacements.items():
                content = re.sub(pattern, replacement, content)
                
            with open(filepath, 'w') as f:
                f.write(content)

print("Done Go source updates")
