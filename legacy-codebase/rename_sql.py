import os
import re

directory = "new-backend/monolith/internal/modules/meal/sql/queries"

replacements = {
    r'\bstudents_cache\b': 'meal.students_view',
    r'\bcafeterias\b': 'meal.cafeterias',
    r'\bmonthly_menus\b': 'meal.monthly_menus',
    r'\bclosed_days\b': 'meal.closed_days',
    r'\breservations\b': 'meal.reservations',
    r'\boutbox_events\b': 'meal.outbox_events',
    r'\bprocessed_events\b': 'meal.processed_events',
    r'\breservation_status_enum\b': 'meal.reservation_status_enum',
    r'\bmeal_time_enum\b': 'meal.meal_time_enum',
    r'\bmenu_type_enum\b': 'meal.menu_type_enum',
    r'\boutbox_status_enum\b': 'meal.outbox_status_enum',
}

for filename in os.listdir(directory):
    if filename.endswith(".sql"):
        filepath = os.path.join(directory, filename)
        with open(filepath, 'r') as f:
            content = f.read()
        
        for pattern, replacement in replacements.items():
            content = re.sub(pattern, replacement, content)
            
        with open(filepath, 'w') as f:
            f.write(content)
print("Done SQL prefixing")
