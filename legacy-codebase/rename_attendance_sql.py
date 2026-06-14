import os
import re

query_dir = "new-backend/monolith/internal/modules/attendance/sql/queries"

replacements = {
    r'\bstudents_cache\b': 'attendance.students_view',
    r'\bcourses_cache\b': 'attendance.courses_view',
    r'\benrollments_cache\b': 'attendance.enrollments_view',
    r'\battendance_sessions\b': 'attendance.attendance_sessions',
    r'\battendance_records\b': 'attendance.attendance_records',
    r'\bacademic_periods\b': 'attendance.academic_periods',
    r'\boutbox_events\b': 'attendance.outbox_events',
    r'\bprocessed_events\b': 'attendance.processed_events',
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

for root, _, files in os.walk(query_dir):
    for file in files:
        if file.endswith('.sql'):
            process_file(os.path.join(root, file))
