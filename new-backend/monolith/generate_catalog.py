import uuid

courses = [
    # Year 1, Sem 1
    ("BB101", "Bilgisayar Bilimine Giriş", 1, 1, 3, 5, 3, 0, "mandatory"),
    ("BB103", "Programlama Temelleri I", 1, 1, 4, 6, 3, 2, "mandatory"),
    ("MAT101", "Kalkülüs I", 1, 1, 4, 6, 4, 0, "mandatory"),
    ("FZK101", "Fizik I", 1, 1, 3, 5, 3, 2, "mandatory"),
    ("ING101", "İngilizce I", 1, 1, 2, 3, 2, 0, "mandatory"),
    ("TUR101", "Türk Dili I", 1, 1, 2, 2, 2, 0, "mandatory"),
    
    # Year 1, Sem 2
    ("BB102", "Programlama Temelleri II", 1, 2, 4, 6, 3, 2, "mandatory"),
    ("BB104", "Ayrık Matematik", 1, 2, 3, 5, 3, 0, "mandatory"),
    ("MAT102", "Kalkülüs II", 1, 2, 4, 6, 4, 0, "mandatory"),
    ("FZK102", "Fizik II", 1, 2, 3, 5, 3, 2, "mandatory"),
    ("ING102", "İngilizce II", 1, 2, 2, 3, 2, 0, "mandatory"),
    ("TUR102", "Türk Dili II", 1, 2, 2, 2, 2, 0, "mandatory"),
    
    # Year 2, Sem 3
    ("BB201", "Veri Yapıları", 2, 3, 4, 6, 3, 2, "mandatory"),
    ("BB203", "Sayısal Mantık Tasarımı", 2, 3, 3, 5, 3, 2, "mandatory"),
    ("BB205", "Nesne Yönelimli Programlama", 2, 3, 3, 5, 2, 2, "mandatory"),
    ("MAT201", "Lineer Cebir", 2, 3, 3, 5, 3, 0, "mandatory"),
    ("ATA101", "Atatürk İlkeleri ve İnkılap Tarihi I", 2, 3, 2, 2, 2, 0, "mandatory"),
    
    # Year 2, Sem 4
    ("BB202", "Algoritma Analizi", 2, 4, 3, 6, 3, 0, "mandatory"),
    ("BB204", "Bilgisayar Mimarisi", 2, 4, 3, 5, 3, 0, "mandatory"),
    ("BB206", "Veritabanı Yönetim Sistemleri", 2, 4, 4, 6, 3, 2, "mandatory"),
    ("BB208", "Olasılık ve İstatistik", 2, 4, 3, 5, 3, 0, "mandatory"),
    ("ATA102", "Atatürk İlkeleri ve İnkılap Tarihi II", 2, 4, 2, 2, 2, 0, "mandatory"),
    
    # Year 3, Sem 5
    ("BB301", "İşletim Sistemleri", 3, 5, 4, 6, 3, 2, "mandatory"),
    ("BB303", "Yazılım Mühendisliği", 3, 5, 3, 5, 3, 0, "mandatory"),
    ("BB305", "Bilgisayar Ağları", 3, 5, 3, 5, 3, 0, "mandatory"),
    ("BB307", "Biçimsel Diller ve Otomata Teorisi", 3, 5, 3, 5, 3, 0, "mandatory"),
    ("BB311", "Yapay Zekaya Giriş (Seçmeli)", 3, 5, 3, 4, 3, 0, "elective"),
    ("BB313", "Web Programlama (Seçmeli)", 3, 5, 3, 4, 2, 2, "elective"),
    
    # Year 3, Sem 6
    ("BB302", "Sistem Programlama", 3, 6, 3, 6, 3, 2, "mandatory"),
    ("BB304", "Gömülü Sistemler", 3, 6, 3, 5, 2, 2, "mandatory"),
    ("BB306", "Mikroişlemciler", 3, 6, 3, 5, 2, 2, "mandatory"),
    ("BB312", "Makine Öğrenmesi (Seçmeli)", 3, 6, 3, 4, 3, 0, "elective"),
    ("BB314", "Mobil Uygulama Geliştirme (Seçmeli)", 3, 6, 3, 4, 2, 2, "elective"),
    ("BB316", "Siber Güvenlik (Seçmeli)", 3, 6, 3, 4, 3, 0, "elective"),
    
    # Year 4, Sem 7
    ("BB401", "Bitirme Projesi I", 4, 7, 2, 6, 1, 2, "mandatory"),
    ("BB403", "Büyük Veri Analizi (Seçmeli)", 4, 7, 3, 5, 3, 0, "elective"),
    ("BB405", "Bulut Bilişim (Seçmeli)", 4, 7, 3, 5, 3, 0, "elective"),
    ("BB407", "Görüntü İşleme (Seçmeli)", 4, 7, 3, 5, 3, 0, "elective"),
    ("BB409", "Derin Öğrenme (Seçmeli)", 4, 7, 3, 5, 3, 0, "elective"),
    
    # Year 4, Sem 8
    ("BB402", "Bitirme Projesi II", 4, 8, 2, 6, 1, 2, "mandatory"),
    ("BB404", "Blokzincir Teknolojileri (Seçmeli)", 4, 8, 3, 5, 3, 0, "elective"),
    ("BB406", "Nesnelerin İnterneti (Seçmeli)", 4, 8, 3, 5, 3, 0, "elective"),
    ("BB408", "İnsan Bilgisayar Etkileşimi (Seçmeli)", 4, 8, 3, 5, 3, 0, "elective"),
    ("BB410", "Dağıtık Sistemler (Seçmeli)", 4, 8, 3, 5, 3, 0, "elective"),
]

sql_commands = []
for c in courses:
    code, name, class_level, semester, credits, ects, theo, lab, ctype = c
    sql_commands.append(
        f"INSERT INTO course_catalog.course_catalog (id, course_code, name, faculty, department, class_level, semester, credits, ects, theoretical_hours, lab_hours, course_type, course_category, education_level, teaching_type, language, status) "
        f"VALUES (gen_random_uuid(), '{code}', '{name}', 'Fen Fakültesi', 'Bilgisayar Bilimi', {class_level}, {semester}, {credits}, {ects}, {theo}, {lab}, '{ctype}', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active') "
        f"ON CONFLICT (course_code) DO NOTHING;"
    )

with open('full_catalog.sql', 'w') as f:
    f.write("\n".join(sql_commands))

print("full_catalog.sql created!")
