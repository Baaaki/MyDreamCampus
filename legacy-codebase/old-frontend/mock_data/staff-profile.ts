// Mock data for staff profile page

export interface EducationInfo {
    id: string;
    degree: string;
    institution: string;
    department: string;
    year: number;
}

export interface Article {
    id: string;
    title: string;
    journal: string;
    year: number;
    authors: string;
    doi?: string;
    journalType?: string;
    domesticInternational?: string;
    publishingMonth?: string;
    issuePageYear?: string;
    language?: string;
    articleType?: string;
}

export interface Bulletin {
    id: string;
    title: string;
    conference: string;
    year: number;
    location: string;
}

export interface Project {
    id: string;
    title: string;
    role: string;
    funder: string;
    startYear: number;
    endYear?: number;
    status: 'ongoing' | 'completed';
}

export interface AwardItem {
    id: string;
    title: string;
    institution: string;
    year: number;
}

export interface Scholarship {
    id: string;
    title: string;
    institution: string;
    year: number;
}

export interface AdminAssignment {
    id: string;
    title: string;
    institution: string;
    startYear: number;
    endYear?: number;
}

export interface StaffProfile {
    id: string;
    title: string;
    firstName: string;
    lastName: string;
    faculty: string;
    department: string;
    email: string;
    phone: string;
    profileImage?: string;
    education: EducationInfo[];
    articles: Article[];
    bulletins: Bulletin[];
    projects: Project[];
    awards: AwardItem[];
    scholarships: Scholarship[];
    adminAssignments: AdminAssignment[];
}

// Her hoca için ayrı profil verisi
export const mockStaffProfiles: Record<string, StaffProfile> = {
    // Ahmet Yılmaz - Bilgisayar Mühendisliği
    '550e8400-e29b-41d4-a716-446655440002': {
        id: '550e8400-e29b-41d4-a716-446655440002',
        title: 'Prof. Dr.',
        firstName: 'Ahmet',
        lastName: 'Yılmaz',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        department: 'BİLGİSAYAR MÜHENDİSLİĞİ BÖLÜMÜ',
        email: 'ahmet.yilmaz@mydreamcampus.edu.tr',
        phone: '+90 532 111 2233',
        profileImage: 'https://i.pravatar.cc/150?img=11',
        education: [
            { id: '1', degree: 'Lisans', institution: 'ODTÜ Mühendislik Fakültesi Bilgisayar Mühendisliği', department: '', year: 1995 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'ODTÜ Fen Bilimleri Enstitüsü Bilgisayar Mühendisliği', department: '', year: 1998 },
            { id: '3', degree: 'Doktora', institution: 'Stanford University Computer Science', department: '', year: 2003 },
            { id: '4', degree: 'Profesör', institution: 'Dokuz Eylül Üniversitesi Mühendislik Fakültesi', department: '', year: 2015 },
        ],
        articles: [
            { id: '1', title: 'Deep Learning Applications in Autonomous Systems', journal: 'IEEE TRANSACTIONS ON NEURAL NETWORKS', year: 2023, authors: 'Yılmaz A., Kaya M.', doi: '10.1109/TNN.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Distributed Computing Architecture for IoT', journal: 'ACM COMPUTING SURVEYS', year: 2022, authors: 'Yılmaz A., Demir B.', doi: '10.1145/cs.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Survey article' },
            { id: '3', title: 'Cloud-Native Application Development Patterns', journal: 'SOFTWARE: PRACTICE AND EXPERIENCE', year: 2021, authors: 'Yılmaz A., Özdemir M., Aydın E.', doi: '10.1002/spe.2021.042', journalType: 'SCI-Expanded', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'Microservices Security Best Practices', conference: 'IEEE International Conference on Cloud Computing', year: 2023, location: 'San Francisco, USA' },
            { id: '2', title: 'Container Orchestration Challenges', conference: 'European Software Engineering Conference', year: 2022, location: 'Amsterdam, Netherlands' },
        ],
        projects: [
            { id: '1', title: 'Akıllı Fabrika Yönetim Sistemi', role: 'Proje Yürütücüsü', funder: 'TÜBİTAK 1001', startYear: 2022, status: 'ongoing' },
            { id: '2', title: 'Bulut Tabanlı Eğitim Platformu', role: 'Proje Yürütücüsü', funder: 'BAP', startYear: 2019, endYear: 2021, status: 'completed' },
        ],
        awards: [
            { id: '1', title: 'IEEE Senior Member', institution: 'IEEE', year: 2020 },
            { id: '2', title: 'En İyi Araştırmacı Ödülü', institution: 'DEÜ Mühendislik Fakültesi', year: 2019 },
        ],
        scholarships: [
            { id: '1', title: 'Fulbright Doktora Bursu', institution: 'Fulbright Commission', year: 2000 },
        ],
        adminAssignments: [
            { id: '1', title: 'Bölüm Başkanı', institution: 'DEÜ Bilgisayar Mühendisliği', startYear: 2021 },
            { id: '2', title: 'Fakülte Kurulu Üyesi', institution: 'DEÜ Mühendislik Fakültesi', startYear: 2018 },
        ],
    },

    // Mehmet Kaya - Bilgisayar Mühendisliği
    '550e8400-e29b-41d4-a716-446655440003': {
        id: '550e8400-e29b-41d4-a716-446655440003',
        title: 'Doç. Dr.',
        firstName: 'Mehmet',
        lastName: 'Kaya',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        department: 'BİLGİSAYAR MÜHENDİSLİĞİ BÖLÜMÜ',
        email: 'mehmet.kaya@mydreamcampus.edu.tr',
        phone: '+90 532 222 3344',
        profileImage: 'https://i.pravatar.cc/150?img=12',
        education: [
            { id: '1', degree: 'Lisans', institution: 'İTÜ Bilgisayar Mühendisliği', department: '', year: 2005 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'MIT Computer Science', department: '', year: 2008 },
            { id: '3', degree: 'Doktora', institution: 'MIT Computer Science', department: '', year: 2012 },
            { id: '4', degree: 'Doçent', institution: 'Dokuz Eylül Üniversitesi', department: '', year: 2020 },
        ],
        articles: [
            { id: '1', title: 'Machine Learning for Natural Language Processing', journal: 'COMPUTATIONAL LINGUISTICS', year: 2023, authors: 'Kaya M., Smith J.', doi: '10.1162/coli.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Transformer Models in Turkish Text Analysis', journal: 'JOURNAL OF ARTIFICIAL INTELLIGENCE RESEARCH', year: 2022, authors: 'Kaya M., Yılmaz A.', doi: '10.1613/jair.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'BERT Applications in Low-Resource Languages', conference: 'ACL Annual Meeting', year: 2023, location: 'Toronto, Canada' },
        ],
        projects: [
            { id: '1', title: 'Türkçe Doğal Dil İşleme Araç Seti', role: 'Proje Yürütücüsü', funder: 'TÜBİTAK 3501', startYear: 2021, status: 'ongoing' },
        ],
        awards: [
            { id: '1', title: 'ACL Best Paper Award', institution: 'Association for Computational Linguistics', year: 2022 },
        ],
        scholarships: [],
        adminAssignments: [
            { id: '1', title: 'Yapay Zeka Laboratuvarı Koordinatörü', institution: 'DEÜ Bilgisayar Mühendisliği', startYear: 2020 },
        ],
    },

    // Ayşe Demir - Matematik
    '550e8400-e29b-41d4-a716-446655440004': {
        id: '550e8400-e29b-41d4-a716-446655440004',
        title: 'Prof. Dr.',
        firstName: 'Ayşe',
        lastName: 'Demir',
        faculty: 'FEN FAKÜLTESİ',
        department: 'MATEMATİK BÖLÜMÜ',
        email: 'ayse.demir@mydreamcampus.edu.tr',
        phone: '+90 532 333 4455',
        profileImage: 'https://i.pravatar.cc/150?img=47',
        education: [
            { id: '1', degree: 'Lisans', institution: 'Boğaziçi Üniversitesi Matematik', department: '', year: 1990 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'Cambridge University Mathematics', department: '', year: 1993 },
            { id: '3', degree: 'Doktora', institution: 'Cambridge University Applied Mathematics', department: '', year: 1997 },
            { id: '4', degree: 'Profesör', institution: 'Dokuz Eylül Üniversitesi Fen Fakültesi', department: '', year: 2010 },
        ],
        articles: [
            { id: '1', title: 'Graph Theory Applications in Network Analysis', journal: 'JOURNAL OF COMBINATORIAL THEORY', year: 2023, authors: 'Demir A., Wilson R.', doi: '10.1016/jct.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Topological Methods in Data Science', journal: 'ADVANCES IN MATHEMATICS', year: 2022, authors: 'Demir A.', doi: '10.1016/aim.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '3', title: 'Algebraic Structures in Cryptography', journal: 'JOURNAL OF ALGEBRA', year: 2021, authors: 'Demir A., Koç Z.', doi: '10.1016/jalg.2021.042', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'Modern Approaches to Graph Coloring', conference: 'International Congress of Mathematicians', year: 2022, location: 'St. Petersburg, Russia' },
        ],
        projects: [
            { id: '1', title: 'Kriptografide Cebirsel Yapılar', role: 'Proje Yürütücüsü', funder: 'TÜBİTAK 1001', startYear: 2020, status: 'ongoing' },
            { id: '2', title: 'Çizge Teorisi ve Ağ Analizi', role: 'Proje Yürütücüsü', funder: 'BAP', startYear: 2017, endYear: 2020, status: 'completed' },
        ],
        awards: [
            { id: '1', title: 'TÜBİTAK Bilim Ödülü', institution: 'TÜBİTAK', year: 2018 },
            { id: '2', title: 'TÜBA Üyeliği', institution: 'Türkiye Bilimler Akademisi', year: 2015 },
        ],
        scholarships: [
            { id: '1', title: 'British Council Scholarship', institution: 'British Council', year: 1991 },
        ],
        adminAssignments: [
            { id: '1', title: 'Fen Fakültesi Dekan Yardımcısı', institution: 'DEÜ Fen Fakültesi', startYear: 2019 },
            { id: '2', title: 'Matematik Bölüm Başkanı', institution: 'DEÜ Fen Fakültesi', startYear: 2015, endYear: 2019 },
        ],
    },

    // Fatma Şahin - Fizik
    '550e8400-e29b-41d4-a716-446655440010': {
        id: '550e8400-e29b-41d4-a716-446655440010',
        title: 'Doç. Dr.',
        firstName: 'Fatma',
        lastName: 'Şahin',
        faculty: 'FEN FAKÜLTESİ',
        department: 'FİZİK BÖLÜMÜ',
        email: 'fatma.sahin@mydreamcampus.edu.tr',
        phone: '+90 532 444 5566',
        profileImage: 'https://i.pravatar.cc/150?img=48',
        education: [
            { id: '1', degree: 'Lisans', institution: 'Hacettepe Üniversitesi Fizik', department: '', year: 2008 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'Max Planck Institute for Physics', department: '', year: 2011 },
            { id: '3', degree: 'Doktora', institution: 'Max Planck Institute for Physics', department: '', year: 2015 },
            { id: '4', degree: 'Doçent', institution: 'Dokuz Eylül Üniversitesi', department: '', year: 2022 },
        ],
        articles: [
            { id: '1', title: 'Quantum Entanglement in Multi-Particle Systems', journal: 'PHYSICAL REVIEW LETTERS', year: 2023, authors: 'Şahin F., Mueller H.', doi: '10.1103/prl.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Dark Matter Detection Methods', journal: 'NATURE PHYSICS', year: 2022, authors: 'Şahin F., CERN Collaboration', doi: '10.1038/nphys.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'CERN LHC Experiments Update', conference: 'European Physical Society Conference', year: 2023, location: 'Geneva, Switzerland' },
        ],
        projects: [
            { id: '1', title: 'CERN ATLAS Deneyi Türkiye Katkısı', role: 'Araştırmacı', funder: 'TÜBİTAK', startYear: 2018, status: 'ongoing' },
        ],
        awards: [
            { id: '1', title: 'L\'Oréal-UNESCO Genç Bilim Kadını Ödülü', institution: 'L\'Oréal-UNESCO', year: 2020 },
        ],
        scholarships: [
            { id: '1', title: 'DAAD Doktora Bursu', institution: 'DAAD', year: 2011 },
        ],
        adminAssignments: [
            { id: '1', title: 'Fizik Bölümü Erasmus Koordinatörü', institution: 'DEÜ Fen Fakültesi', startYear: 2021 },
        ],
    },

    // Mustafa Özdemir - Elektrik-Elektronik Mühendisliği
    '550e8400-e29b-41d4-a716-446655440011': {
        id: '550e8400-e29b-41d4-a716-446655440011',
        title: 'Dr. Öğr. Üyesi',
        firstName: 'Mustafa',
        lastName: 'Özdemir',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        department: 'ELEKTRİK-ELEKTRONİK MÜHENDİSLİĞİ BÖLÜMÜ',
        email: 'mustafa.ozdemir@mydreamcampus.edu.tr',
        phone: '+90 532 555 6677',
        profileImage: 'https://i.pravatar.cc/150?img=13',
        education: [
            { id: '1', degree: 'Lisans', institution: 'Yıldız Teknik Üniversitesi Elektrik Mühendisliği', department: '', year: 2010 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'TU Munich Electrical Engineering', department: '', year: 2013 },
            { id: '3', degree: 'Doktora', institution: 'TU Munich Electrical Engineering', department: '', year: 2018 },
        ],
        articles: [
            { id: '1', title: 'Power Electronics for Renewable Energy Systems', journal: 'IEEE TRANSACTIONS ON POWER ELECTRONICS', year: 2023, authors: 'Özdemir M., Schmidt K.', doi: '10.1109/TPEL.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Smart Grid Control Systems', journal: 'ELECTRIC POWER SYSTEMS RESEARCH', year: 2022, authors: 'Özdemir M.', doi: '10.1016/epsr.2022.015', journalType: 'SCI-Expanded', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'Electric Vehicle Charging Infrastructure', conference: 'IEEE PowerTech Conference', year: 2023, location: 'Belgrade, Serbia' },
        ],
        projects: [
            { id: '1', title: 'Akıllı Şebeke Yönetim Sistemi', role: 'Proje Yürütücüsü', funder: 'EÜAŞ', startYear: 2022, status: 'ongoing' },
        ],
        awards: [],
        scholarships: [
            { id: '1', title: 'DAAD Yüksek Lisans Bursu', institution: 'DAAD', year: 2011 },
        ],
        adminAssignments: [
            { id: '1', title: 'Güç Elektroniği Lab. Sorumlusu', institution: 'DEÜ Elektrik-Elektronik Müh.', startYear: 2020 },
        ],
    },

    // Elif Aydın - Bilgisayar Mühendisliği
    '550e8400-e29b-41d4-a716-446655440012': {
        id: '550e8400-e29b-41d4-a716-446655440012',
        title: 'Dr. Öğr. Üyesi',
        firstName: 'Elif',
        lastName: 'Aydın',
        faculty: 'MÜHENDİSLİK FAKÜLTESİ',
        department: 'BİLGİSAYAR MÜHENDİSLİĞİ BÖLÜMÜ',
        email: 'elif.aydin@mydreamcampus.edu.tr',
        phone: '+90 532 666 7788',
        profileImage: 'https://i.pravatar.cc/150?img=49',
        education: [
            { id: '1', degree: 'Lisans', institution: 'Bilkent Üniversitesi Bilgisayar Mühendisliği', department: '', year: 2014 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'ETH Zurich Computer Science', department: '', year: 2017 },
            { id: '3', degree: 'Doktora', institution: 'ETH Zurich Computer Science', department: '', year: 2021 },
        ],
        articles: [
            { id: '1', title: 'Federated Learning for Privacy-Preserving AI', journal: 'JOURNAL OF MACHINE LEARNING RESEARCH', year: 2023, authors: 'Aydın E., Fischer T.', doi: '10.5555/jmlr.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Edge Computing in Healthcare Applications', journal: 'IEEE INTERNET OF THINGS JOURNAL', year: 2022, authors: 'Aydın E., Yılmaz A.', doi: '10.1109/JIOT.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'Privacy in Distributed Machine Learning', conference: 'NeurIPS', year: 2022, location: 'New Orleans, USA' },
        ],
        projects: [
            { id: '1', title: 'Gizlilik Korumalı Makine Öğrenmesi', role: 'Proje Yürütücüsü', funder: 'TÜBİTAK 3501', startYear: 2023, status: 'ongoing' },
        ],
        awards: [
            { id: '1', title: 'Google PhD Fellowship', institution: 'Google', year: 2019 },
        ],
        scholarships: [],
        adminAssignments: [],
    },

    // Zeynep Koç - Kimya
    '550e8400-e29b-41d4-a716-446655440013': {
        id: '550e8400-e29b-41d4-a716-446655440013',
        title: 'Prof. Dr.',
        firstName: 'Zeynep',
        lastName: 'Koç',
        faculty: 'FEN FAKÜLTESİ',
        department: 'KİMYA BÖLÜMÜ',
        email: 'zeynep.koc@mydreamcampus.edu.tr',
        phone: '+90 532 777 8899',
        profileImage: 'https://i.pravatar.cc/150?img=50',
        education: [
            { id: '1', degree: 'Lisans', institution: 'Ege Üniversitesi Kimya', department: '', year: 1998 },
            { id: '2', degree: 'Yüksek Lisans', institution: 'Oxford University Chemistry', department: '', year: 2001 },
            { id: '3', degree: 'Doktora', institution: 'Oxford University Organic Chemistry', department: '', year: 2005 },
            { id: '4', degree: 'Profesör', institution: 'Dokuz Eylül Üniversitesi', department: '', year: 2018 },
        ],
        articles: [
            { id: '1', title: 'Green Chemistry Approaches in Pharmaceutical Synthesis', journal: 'JOURNAL OF THE AMERICAN CHEMICAL SOCIETY', year: 2023, authors: 'Koç Z., Brown M.', doi: '10.1021/jacs.2023.001', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '2', title: 'Sustainable Catalysis for Industrial Applications', journal: 'NATURE CHEMISTRY', year: 2022, authors: 'Koç Z.', doi: '10.1038/nchem.2022.015', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
            { id: '3', title: 'Metal-Organic Frameworks for Drug Delivery', journal: 'ANGEWANDTE CHEMIE', year: 2021, authors: 'Koç Z., Demir A.', doi: '10.1002/anie.2021.042', journalType: 'SCI', domesticInternational: 'YURTDIŞI', language: 'English', articleType: 'Research article' },
        ],
        bulletins: [
            { id: '1', title: 'Advances in Green Chemistry', conference: 'IUPAC World Chemistry Congress', year: 2023, location: 'The Hague, Netherlands' },
            { id: '2', title: 'Nanomaterials for Environmental Remediation', conference: 'ACS National Meeting', year: 2022, location: 'Chicago, USA' },
        ],
        projects: [
            { id: '1', title: 'Yeşil Kimya ile İlaç Sentezi', role: 'Proje Yürütücüsü', funder: 'TÜBİTAK 1001', startYear: 2021, status: 'ongoing' },
            { id: '2', title: 'Sürdürülebilir Kataliz Yöntemleri', role: 'Proje Yürütücüsü', funder: 'AB Horizon 2020', startYear: 2018, endYear: 2022, status: 'completed' },
        ],
        awards: [
            { id: '1', title: 'TÜBA-GEBİP Ödülü', institution: 'Türkiye Bilimler Akademisi', year: 2012 },
            { id: '2', title: 'Türkiye Kimya Derneği Bilim Ödülü', institution: 'TKD', year: 2020 },
        ],
        scholarships: [
            { id: '1', title: 'Chevening Scholarship', institution: 'British Council', year: 1999 },
        ],
        adminAssignments: [
            { id: '1', title: 'Kimya Bölüm Başkanı', institution: 'DEÜ Fen Fakültesi', startYear: 2020 },
            { id: '2', title: 'Fen Fakültesi Yönetim Kurulu Üyesi', institution: 'DEÜ Fen Fakültesi', startYear: 2018 },
        ],
    },
};

// Varsayılan profil (geriye dönük uyumluluk için)
export const mockStaffProfile: StaffProfile = mockStaffProfiles['550e8400-e29b-41d4-a716-446655440004'];

// ID'ye göre profil getirme fonksiyonu
export const getStaffProfileById = (id: string): StaffProfile | null => {
    return mockStaffProfiles[id] || null;
};
