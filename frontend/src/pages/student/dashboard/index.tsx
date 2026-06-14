
import { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  User,
  Quote,
  Megaphone,
  Wifi,
  Bell,
  Mail,
  ExternalLink,
  Sparkles,
  Calendar,
} from 'lucide-react';
import { studentApi } from '@/lib/api-client';

interface StudentInfo {
  name: string;
  studentId: string;
  department: string;
  faculty: string;
  advisorName?: string;
}

const quotes = [
  { text: 'Başarı, her gün tekrarlanan küçük çabaların toplamıdır.', author: 'Robert Collier' },
  { text: 'Eğitim en güçlü silahtır. Dünyayı değiştirmek için kullanabilirsiniz.', author: 'Nelson Mandela' },
  { text: 'Öğrenmenin sınırı yoktur.', author: 'Konfüçyüs' },
  { text: 'Bugün yapabileceğini yarına bırakma.', author: 'Benjamin Franklin' },
  { text: 'Bilgi güçtür.', author: 'Francis Bacon' },
  { text: 'Başarısızlık, başarıya giden yolda sadece bir duraktır.', author: 'Zig Ziglar' },
  { text: 'Hayatta en hakiki mürşit ilimdir.', author: 'Mustafa Kemal Atatürk' },
  { text: 'Gelecek, bugünden başlar.', author: 'Malcolm X' },
  { text: 'Öğrenmek, bir hazineye sahip olmaktır.', author: 'Çin Atasözü' },
  { text: 'Azim ve kararlılık her şeyi başarır.', author: 'Benjamin Disraeli' },
  { text: 'Düşünceleriniz kaderinizi belirler.', author: 'Lao Tzu' },
  { text: 'Başlamak, bitirmenin yarısıdır.', author: 'Aristoteles' },
];

const announcements = [
  {
    id: 1,
    title: 'Bahar Dönemi Ders Kayıtları Başladı',
    date: '2026-01-15',
    isNew: true,
    category: 'Akademik',
  },
  {
    id: 2,
    title: 'Kütüphane Çalışma Saatleri Güncellendi',
    date: '2026-01-14',
    isNew: true,
    category: 'Duyuru',
  },
  {
    id: 3,
    title: 'Kariyer Günleri Etkinliği - 20 Ocak',
    date: '2026-01-12',
    isNew: false,
    category: 'Etkinlik',
  },
];

const itAnnouncements = [
  {
    id: 1,
    title: 'E-posta sistemi bakım çalışması',
    date: '2026-01-16',
    description: '16 Ocak Cumartesi 02:00-06:00 arası e-posta sistemi bakımda olacaktır.',
  },
  {
    id: 2,
    title: 'VPN erişimi güncellendi',
    date: '2026-01-10',
    description: 'Kampüs dışından VPN bağlantısı için yeni ayarlar yayınlandı.',
  },
];

const messages = [
  {
    id: 1,
    from: 'Danışman',
    subject: 'Ders Seçimi Hakkında',
    date: '2026-01-14',
    unread: true,
  },
  {
    id: 2,
    from: 'Öğrenci İşleri',
    subject: 'Belge Talebi Onaylandı',
    date: '2026-01-13',
    unread: false,
  },
];

export default function StudentDashboardPage() {
  const [studentInfo, setStudentInfo] = useState<StudentInfo>({
    name: '',
    studentId: '',
    department: '',
    faculty: '',
  });
  const [loading, setLoading] = useState(true);
  const [currentQuote, setCurrentQuote] = useState(quotes[0]);

  useEffect(() => {
    const fetchStudentInfo = async () => {
      try {
        // Get user from localStorage
        const userStr = localStorage.getItem('user');
        if (userStr) {
          const user = JSON.parse(userStr);
          // Fetch student details from API using user ID
          const response = await studentApi.get(`${user.id}`).json<{
            id: string;
            student_number: string;
            first_name: string;
            last_name: string;
            email: string;
            faculty: string;
            department: string;
            advisor_name?: string;
          }>();

          setStudentInfo({
            name: `${response.first_name} ${response.last_name}`,
            studentId: response.student_number,
            department: response.department,
            faculty: response.faculty,
            advisorName: response.advisor_name,
          });
        }
      } catch (error) {
        console.error('Failed to fetch student info:', error);
        // Fallback to localStorage user info
        const userStr = localStorage.getItem('user');
        if (userStr) {
          const user = JSON.parse(userStr);
          setStudentInfo({
            name: user.email?.split('@')[0] || 'Öğrenci',
            studentId: '-',
            department: user.department || '-',
            faculty: '-',
          });
        }
      } finally {
        setLoading(false);
      }
    };

    fetchStudentInfo();
  }, []);

  // Rotate quotes every 5 minutes
  useEffect(() => {
    const getRandomQuote = () => {
      const randomIndex = Math.floor(Math.random() * quotes.length);
      setCurrentQuote(quotes[randomIndex]);
    };

    // Set initial random quote
    getRandomQuote();

    // Change quote every 5 minutes (300000ms)
    const interval = setInterval(getRandomQuote, 5 * 60 * 1000);

    return () => clearInterval(interval);
  }, []);

  const currentHour = new Date().getHours();
  const greeting = currentHour < 12 ? 'Günaydın' : currentHour < 18 ? 'İyi Günler' : 'İyi Akşamlar';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Öğrenci Portalı</h1>
          <p className="text-gray-600 dark:text-gray-400">{new Date().toLocaleDateString('tr-TR', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}</p>
        </div>
      </div>

      {/* Grid Layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Merhaba - Welcome Card */}
        <Card className="bg-gradient-to-br from-emerald-500 to-emerald-600 text-white border-0">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-white">
              <User className="h-5 w-5" />
              {greeting}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold mb-2">
              {loading ? 'Yükleniyor...' : `Sayın ${studentInfo.name}`}
            </p>
            <div className="space-y-1 text-emerald-100 text-sm">
              <p>Öğrenci No: {studentInfo.studentId || '-'}</p>
              <p>{studentInfo.department || '-'}</p>
              <p>{studentInfo.faculty || '-'}</p>
              {studentInfo.advisorName && <p>Danışman: {studentInfo.advisorName}</p>}
            </div>
          </CardContent>
        </Card>

        {/* Günün Sözü - Quote of the Day */}
        <Card className="bg-gradient-to-br from-blue-500 to-blue-600 text-white border-0">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-white">
              <Quote className="h-5 w-5" />
              Günün Sözü
            </CardTitle>
          </CardHeader>
          <CardContent>
            <blockquote className="text-lg italic mb-3">
              "{currentQuote.text}"
            </blockquote>
            <p className="text-blue-100 text-right font-medium">— {currentQuote.author}</p>
          </CardContent>
        </Card>

        {/* Duyurular - Announcements */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <Megaphone className="h-5 w-5 text-orange-500" />
              Kurumsal Duyurular
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {announcements.map((item) => (
                <div
                  key={item.id}
                  className="flex items-start gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors cursor-pointer"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      {item.isNew && (
                        <Badge className="bg-red-500 text-white text-[10px] px-1.5 py-0">YENİ</Badge>
                      )}
                      <Badge variant="outline" className="text-[10px]">{item.category}</Badge>
                    </div>
                    <p className="text-sm font-medium text-gray-900 dark:text-white">{item.title}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      <Calendar className="h-3 w-3 inline mr-1" />
                      {new Date(item.date).toLocaleDateString('tr-TR')}
                    </p>
                  </div>
                  <ExternalLink className="h-4 w-4 text-gray-400" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Bilgi İşlem Duyuruları - IT Announcements */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <Bell className="h-5 w-5 text-purple-500" />
              Bilgi İşlem Duyuruları
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {itAnnouncements.map((item) => (
                <div
                  key={item.id}
                  className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800"
                >
                  <p className="text-sm font-medium text-gray-900 dark:text-white mb-1">{item.title}</p>
                  <p className="text-xs text-gray-600 dark:text-gray-400">{item.description}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-500 mt-2">
                    <Calendar className="h-3 w-3 inline mr-1" />
                    {new Date(item.date).toLocaleDateString('tr-TR')}
                  </p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Mesajlar - Messages */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <Mail className="h-5 w-5 text-indigo-500" />
              Mesajlar
              {messages.filter(m => m.unread).length > 0 && (
                <Badge className="bg-red-500 text-white text-xs">
                  {messages.filter(m => m.unread).length} yeni
                </Badge>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {messages.map((msg) => (
                <div
                  key={msg.id}
                  className={`flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors ${
                    msg.unread 
                      ? 'bg-indigo-50 dark:bg-indigo-900/30 hover:bg-indigo-100 dark:hover:bg-indigo-900/50' 
                      : 'bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700'
                  }`}
                >
                  <div className={`w-2 h-2 rounded-full ${msg.unread ? 'bg-indigo-500' : 'bg-transparent'}`} />
                  <div className="flex-1">
                    <p className={`text-sm ${msg.unread ? 'font-semibold' : 'font-medium'} text-gray-900 dark:text-white`}>
                      {msg.subject}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {msg.from} • {new Date(msg.date).toLocaleDateString('tr-TR')}
                    </p>
                  </div>
                </div>
              ))}
              {messages.length === 0 && (
                <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
                  Yeni mesajınız yok.
                </p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Kablosuz Ağ Bağlantısı - WiFi Connection */}
        <Card className="bg-gradient-to-br from-cyan-500 to-cyan-600 text-white border-0">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-white">
              <Wifi className="h-5 w-5" />
              Kablosuz Ağ Bağlantısı
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <p className="text-cyan-100 text-sm">
                Kampüs WiFi ağına bağlanmak için <strong>eduroam</strong> ağını seçin.
              </p>
              <div className="bg-white/20 rounded-lg p-3 text-sm">
                <p><strong>Ağ Adı:</strong> eduroam</p>
                <p><strong>Kullanıcı Adı:</strong> ogrenci_no@ogrenci.edu.tr</p>
                <p><strong>Şifre:</strong> E-posta şifreniz</p>
              </div>
              <button className="w-full bg-white/20 hover:bg-white/30 text-white py-2 px-4 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2">
                <Sparkles className="h-4 w-4" />
                Detaylı Kurulum Kılavuzu
              </button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
