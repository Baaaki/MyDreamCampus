// Mock data for teacher's current semester courses

export interface TeacherCourse {
    id: string;
    course_code: string;
    name: string;
    faculty: string;
    department: string;
    semester: string; // e.g., "2025-2026 Güz"
    credits: number;
    theoretical_hours: number;
    lab_hours: number;
    student_count: number;
    schedule: {
        day: string;
        time: string;
        room: string;
    }[];
}

export interface CurrentSemester {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_active: boolean;
}

export const mockCurrentSemester: CurrentSemester = {
    id: 'sem-2025-2026-spring',
    name: '2025-2026 Bahar',
    start_date: '2026-02-03',
    end_date: '2026-06-15',
    is_active: true,
};

// Hocanın güncel dönemde kayıtlı olduğu dersler
export const mockTeacherCourses: TeacherCourse[] = [
    {
        id: 'tc-bil2011',
        course_code: 'BİL 2011',
        name: 'Algoritmalar ve Veri Yapıları',
        faculty: 'Fen Fakültesi',
        department: 'Bilgisayar Bilimleri',
        semester: '2025-2026 Bahar',
        credits: 6,
        theoretical_hours: 2,
        lab_hours: 2,
        student_count: 45,
        schedule: [
            { day: 'Pazartesi', time: '09:00-10:50', room: 'A-201' },
            { day: 'Çarşamba', time: '13:00-14:50', room: 'Lab-B3' },
        ],
    },
    {
        id: 'tc-bil2015',
        course_code: 'BİL 2015',
        name: 'Nesneye Yönelik Analiz ve Tasarım',
        faculty: 'Fen Fakültesi',
        department: 'Bilgisayar Bilimleri',
        semester: '2025-2026 Bahar',
        credits: 5,
        theoretical_hours: 2,
        lab_hours: 2,
        student_count: 38,
        schedule: [
            { day: 'Salı', time: '10:00-11:50', room: 'A-305' },
            { day: 'Perşembe', time: '14:00-15:50', room: 'Lab-B2' },
        ],
    },
    {
        id: 'tc-bil3021',
        course_code: 'BİL 3021',
        name: 'Yapay Zeka',
        faculty: 'Fen Fakültesi',
        department: 'Bilgisayar Bilimleri',
        semester: '2025-2026 Bahar',
        credits: 5,
        theoretical_hours: 3,
        lab_hours: 0,
        student_count: 52,
        schedule: [
            { day: 'Cuma', time: '09:00-11:50', room: 'A-101' },
        ],
    },
    {
        id: 'tc-bil4005',
        course_code: 'BİL 4005',
        name: 'Yazılım Mühendisliği',
        faculty: 'Fen Fakültesi',
        department: 'Bilgisayar Bilimleri',
        semester: '2025-2026 Bahar',
        credits: 6,
        theoretical_hours: 3,
        lab_hours: 2,
        student_count: 35,
        schedule: [
            { day: 'Pazartesi', time: '13:00-15:50', room: 'A-202' },
            { day: 'Çarşamba', time: '09:00-10:50', room: 'Lab-B1' },
        ],
    },
];
