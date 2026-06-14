// Mock data for administrative staff

export interface AdminStaffProfile {
    id: string;
    title: string;
    firstName: string;
    lastName: string;
    faculty: string;
    department?: string;
    email: string;
    phone: string;
    profileImage?: string;
    position: string; // Görev pozisyonu
    jobDescription: string; // Yaptığı iş
    responsibilities: string[]; // Sorumlulukları
    workingHours: string; // Çalışma saatleri
    officeLocation: string; // Ofis konumu
    startDate: string; // Göreve başlama tarihi
}

// İdari personel listesi
export const mockAdminStaff = [
    {
        id: 'admin-001',
        email: 'ali.vural@mydreamcampus.edu.tr',
        first_name: 'Ali',
        last_name: 'Vural',
        role: 'admin',
        faculty: 'Mühendislik Fakültesi',
        phone: '+90 232 301 1001',
        office_location: 'Mühendislik Fakültesi Dekanlık Binası',
        status: 'active',
        position: 'Fakülte Sekreteri',
    },
    {
        id: 'admin-002',
        email: 'zehra.aksoy@mydreamcampus.edu.tr',
        first_name: 'Zehra',
        last_name: 'Aksoy',
        role: 'admin',
        faculty: 'Mühendislik Fakültesi',
        phone: '+90 232 301 1002',
        office_location: 'Mühendislik Fakültesi Öğrenci İşleri',
        status: 'active',
        position: 'Öğrenci İşleri Şefi',
    },
    {
        id: 'admin-003',
        email: 'hasan.yildiz@mydreamcampus.edu.tr',
        first_name: 'Hasan',
        last_name: 'Yıldız',
        role: 'admin',
        faculty: 'Fen Fakültesi',
        phone: '+90 232 301 2001',
        office_location: 'Fen Fakültesi Dekanlık Binası',
        status: 'active',
        position: 'Fakülte Sekreteri',
    },
    {
        id: 'admin-004',
        email: 'ayten.kara@mydreamcampus.edu.tr',
        first_name: 'Ayten',
        last_name: 'Kara',
        role: 'admin',
        faculty: 'Fen Fakültesi',
        phone: '+90 232 301 2002',
        office_location: 'Fen Fakültesi Öğrenci İşleri',
        status: 'active',
        position: 'Öğrenci İşleri Memuru',
    },
];

// İdari personel profil detayları
export const mockAdminStaffProfiles: Record<string, AdminStaffProfile> = {
    'admin-001': {
        id: 'admin-001',
        title: '',
        firstName: 'Ali',
        lastName: 'Vural',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        email: 'ali.vural@mydreamcampus.edu.tr',
        phone: '+90 232 301 1001',
        profileImage: 'https://i.pravatar.cc/150?img=60',
        position: 'Fakülte Sekreteri',
        jobDescription: 'Mühendislik Fakültesi idari işlerinin yönetimi ve koordinasyonu. Dekanlık ile bölümler arası iletişimin sağlanması, fakülte bütçe ve kaynak yönetiminin takibi.',
        responsibilities: [
            'Fakülte dekanlığının idari işlerini yürütmek',
            'Akademik ve idari personelin özlük işlemlerini takip etmek',
            'Fakülte bütçesinin hazırlanmasına katkıda bulunmak',
            'Dekanlık yazışmalarını koordine etmek',
            'Fakülte kurulu ve yönetim kurulu toplantılarını organize etmek',
            'Taşınır mal işlemlerini yürütmek',
        ],
        workingHours: 'Pazartesi - Cuma: 08:30 - 17:30',
        officeLocation: 'Mühendislik Fakültesi Dekanlık Binası, Kat: 1, Oda: 101',
        startDate: '2015-03-01',
    },
    'admin-002': {
        id: 'admin-002',
        title: '',
        firstName: 'Zehra',
        lastName: 'Aksoy',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        email: 'zehra.aksoy@mydreamcampus.edu.tr',
        phone: '+90 232 301 1002',
        profileImage: 'https://i.pravatar.cc/150?img=45',
        position: 'Öğrenci İşleri Şefi',
        jobDescription: 'Mühendislik Fakültesi öğrenci işleri biriminin yönetimi. Öğrenci kayıt, kabul ve mezuniyet işlemlerinin koordinasyonu.',
        responsibilities: [
            'Öğrenci kayıt ve kabul işlemlerini yürütmek',
            'Mezuniyet belgesi ve diploma işlemlerini takip etmek',
            'Öğrenci disiplin işlemlerini yürütmek',
            'Yatay ve dikey geçiş işlemlerini koordine etmek',
            'Öğrenci belgesi ve transkript taleplerini karşılamak',
            'Staj koordinasyonunu sağlamak',
            'Öğrenci danışmanlık hizmetlerine destek vermek',
        ],
        workingHours: 'Pazartesi - Cuma: 09:00 - 17:00',
        officeLocation: 'Mühendislik Fakültesi Öğrenci İşleri, Kat: Zemin, Oda: Z-05',
        startDate: '2018-09-15',
    },
    'admin-003': {
        id: 'admin-003',
        title: '',
        firstName: 'Hasan',
        lastName: 'Yıldız',
        faculty: 'FEN FAKÜLTESİ',
        email: 'hasan.yildiz@mydreamcampus.edu.tr',
        phone: '+90 232 301 2001',
        profileImage: 'https://i.pravatar.cc/150?img=67',
        position: 'Fakülte Sekreteri',
        jobDescription: 'Fen Fakültesi genel idari işlerinin yönetimi. Akademik kadro ve bütçe planlaması, fakülte stratejik planlamasına katkı.',
        responsibilities: [
            'Fakülte idari birimlerini koordine etmek',
            'Akademik personel atama ve yükseltme işlemlerini takip etmek',
            'Fakülte kalite güvence süreçlerine katkıda bulunmak',
            'Resmi yazışmaları yürütmek',
            'Fakülte etkinliklerini organize etmek',
            'Laboratuvar ve araştırma altyapı ihtiyaçlarını koordine etmek',
        ],
        workingHours: 'Pazartesi - Cuma: 08:30 - 17:30',
        officeLocation: 'Fen Fakültesi Dekanlık Binası, Kat: 2, Oda: 201',
        startDate: '2012-06-01',
    },
    'admin-004': {
        id: 'admin-004',
        title: '',
        firstName: 'Ayten',
        lastName: 'Kara',
        faculty: 'FEN FAKÜLTESİ',
        email: 'ayten.kara@mydreamcampus.edu.tr',
        phone: '+90 232 301 2002',
        profileImage: 'https://i.pravatar.cc/150?img=44',
        position: 'Öğrenci İşleri Memuru',
        jobDescription: 'Fen Fakültesi öğrenci işleri işlemlerinin yürütülmesi. Öğrenci belge ve evrak işlemlerinin takibi.',
        responsibilities: [
            'Öğrenci belgesi taleplerini karşılamak',
            'Ders kayıt işlemlerine destek vermek',
            'Not girişi ve transkript işlemlerini takip etmek',
            'Öğrenci dosyalarını düzenlemek ve arşivlemek',
            'Mezuniyet başvurularını almak ve işlemek',
            'Yaz okulu kayıt işlemlerini yürütmek',
        ],
        workingHours: 'Pazartesi - Cuma: 09:00 - 17:00',
        officeLocation: 'Fen Fakültesi Öğrenci İşleri, Kat: Zemin, Oda: Z-02',
        startDate: '2020-02-01',
    },
};

// ID'ye göre idari personel profili getirme
export const getAdminStaffProfileById = (id: string): AdminStaffProfile | null => {
    return mockAdminStaffProfiles[id] || null;
};
