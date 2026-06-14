import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { attendanceApiSafe } from '@/lib/api-client';
import type { AdminSessionsResponse, AdminSessionItem } from '@/lib/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import {
  ChevronLeft,
  ChevronRight,
  Calendar,
} from 'lucide-react';
import { catalogService } from '@/lib/services/catalog-service';
import { mockFaculties, mockCourseCatalog } from '@/mock_data/catalog';
import { mockAdminSessionsResponse } from '@/mock_data/admin_attendance';

function getMonthRange(year: number, month: number) {
  const start = new Date(year, month, 1);
  const end = new Date(year, month + 1, 0);
  return {
    start_date: formatDate(start),
    end_date: formatDate(end),
  };
}

function formatDate(d: Date) {
  return d.toISOString().split('T')[0];
}

function getDaysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate();
}

function getFirstDayOfMonth(year: number, month: number) {
  // 0=Sunday, convert so Monday=0
  const day = new Date(year, month, 1).getDay();
  return day === 0 ? 6 : day - 1;
}

const MONTH_NAMES = [
  'Ocak', 'Şubat', 'Mart', 'Nisan', 'Mayıs', 'Haziran',
  'Temmuz', 'Ağustos', 'Eylül', 'Ekim', 'Kasım', 'Aralık',
];

const DAY_NAMES = ['Pzt', 'Sal', 'Çar', 'Per', 'Cum', 'Cmt', 'Paz'];

export default function AdminAttendancePage() {
  const navigate = useNavigate();
  const today = new Date();
  const [year, setYear] = useState(today.getFullYear());
  const [month, setMonth] = useState(today.getMonth());
  const [selectedDate, setSelectedDate] = useState<string | null>(null);

  const { start_date, end_date } = getMonthRange(year, month);

  const [useMockData, setUseMockData] = useState(false);

  const { data: apiData, isLoading } = useQuery({
    queryKey: ['admin-attendance-sessions', start_date, end_date],
    queryFn: () =>
      attendanceApiSafe
        .get('admin/sessions', { searchParams: { start_date, end_date } })
        .json<AdminSessionsResponse>(),
    enabled: !useMockData,
  });

  const data = useMockData ? mockAdminSessionsResponse : apiData;

  const [facultyFilter, setFacultyFilter] = useState('');
  const [departmentFilter, setDepartmentFilter] = useState('');

  // Fetch courses for the selected department/faculty to filter sessions
  const { data: coursesData } = useQuery({
    queryKey: ['courses-for-attendance-filter', facultyFilter, departmentFilter],
    queryFn: () => catalogService.listCourses({ faculty: facultyFilter, department: departmentFilter, limit: 1000 }),
    enabled: !!facultyFilter && !useMockData,
  });

  const validCourseCodes = useMemo(() => {
    if (useMockData) {
      if (!facultyFilter) return new Set<string>();
      return new Set(mockCourseCatalog
        .filter(c => c.faculty === facultyFilter && (departmentFilter ? c.department === departmentFilter : true))
        .map(c => c.course_code)
      );
    }
    if (!coursesData?.courses) return new Set<string>();
    return new Set(coursesData.courses.map(c => c.course_code));
  }, [coursesData, useMockData, facultyFilter, departmentFilter]);

  // Group sessions by date
  const sessionsByDate = useMemo(() => {
    const map: Record<string, AdminSessionItem[]> = {};
    if (data?.sessions) {
      for (const session of data.sessions) {
        if (facultyFilter && !validCourseCodes.has(session.course_code)) {
          continue;
        }

        if (!map[session.session_date]) {
          map[session.session_date] = [];
        }
        map[session.session_date].push(session);
      }
    }
    return map;
  }, [data, facultyFilter, validCourseCodes]);

  const selectedSessions = selectedDate ? sessionsByDate[selectedDate] ?? [] : [];

  const daysInMonth = getDaysInMonth(year, month);
  const firstDay = getFirstDayOfMonth(year, month);

  const prevMonth = () => {
    if (month === 0) {
      setMonth(11);
      setYear(year - 1);
    } else {
      setMonth(month - 1);
    }
    setSelectedDate(null);
  };

  const nextMonth = () => {
    if (month === 11) {
      setMonth(0);
      setYear(year + 1);
    } else {
      setMonth(month + 1);
    }
    setSelectedDate(null);
  };

  const goToToday = () => {
    setYear(today.getFullYear());
    setMonth(today.getMonth());
    setSelectedDate(formatDate(today));
  };

  // Build calendar grid
  const calendarCells: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) calendarCells.push(null);
  for (let d = 1; d <= daysInMonth; d++) calendarCells.push(d);

  const todayStr = formatDate(today);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Yoklama Takvimi
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Aylık yoklama oturumlarını görüntüleyin
          </p>
        </div>
        <div className="flex items-center gap-2 rounded border px-3 py-1.5 shadow-sm dark:border-gray-800 bg-white dark:bg-gray-900">
          <label htmlFor="mock-toggle" className="text-xs font-semibold text-gray-700 dark:text-gray-300 select-none cursor-pointer">
            Test Modu (Mock Veri)
          </label>
          <input 
            id="mock-toggle" 
            type="checkbox" 
            className="cursor-pointer rounded accent-indigo-600"
            checked={useMockData} 
            onChange={(e) => setUseMockData(e.target.checked)} 
          />
        </div>
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-end rounded-lg border bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
        <div className="grid flex-1 grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Fakülte</label>
            <select
              className="flex h-10 w-full items-center justify-between rounded-md border border-gray-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:ring-offset-gray-950 dark:placeholder:text-gray-400 dark:focus:ring-gray-300"
              value={facultyFilter}
              onChange={(e) => {
                setFacultyFilter(e.target.value);
                setDepartmentFilter('');
              }}
            >
              <option value="">Tüm Fakülteler</option>
              {mockFaculties.map((f) => (
                <option key={f.id} value={f.name}>
                  {f.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Bölüm</label>
            <select
              className="flex h-10 w-full items-center justify-between rounded-md border border-gray-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:ring-offset-gray-950 dark:placeholder:text-gray-400 dark:focus:ring-gray-300"
              value={departmentFilter}
              onChange={(e) => setDepartmentFilter(e.target.value)}
              disabled={!facultyFilter}
            >
              <option value="">Tüm Bölümler</option>
              {mockFaculties.find((f) => f.name === facultyFilter)?.departments.map((d) => (
                <option key={d.id} value={d.name}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-6">
        {/* Calendar */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="flex items-center gap-2">
              <Calendar className="h-5 w-5" />
              {MONTH_NAMES[month]} {year}
            </CardTitle>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={goToToday}>
                Bugün
              </Button>
              <Button variant="outline" size="icon" onClick={prevMonth}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="icon" onClick={nextMonth}>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {/* Day headers */}
            <div className="grid grid-cols-7 mb-1">
              {DAY_NAMES.map((d) => (
                <div
                  key={d}
                  className="text-center text-xs font-medium text-gray-500 dark:text-gray-400 py-2"
                >
                  {d}
                </div>
              ))}
            </div>

            {/* Calendar grid */}
            <div className="grid grid-cols-7 gap-1">
              {calendarCells.map((day, idx) => {
                if (day === null) {
                  return <div key={`empty-${idx}`} className="h-20" />;
                }

                const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
                const sessions = sessionsByDate[dateStr];
                const count = sessions?.length ?? 0;
                const isToday = dateStr === todayStr;
                const isSelected = dateStr === selectedDate;

                return (
                  <button
                    key={dateStr}
                    onClick={() => setSelectedDate(dateStr)}
                    className={`h-20 rounded-lg border p-1.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-gray-800 ${
                      isSelected
                        ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/30 dark:border-indigo-400'
                        : 'border-gray-200 dark:border-gray-700'
                    } ${isToday ? 'ring-2 ring-indigo-300 dark:ring-indigo-600' : ''}`}
                  >
                    <span
                      className={`text-sm font-medium ${
                        isToday
                          ? 'text-indigo-600 dark:text-indigo-400'
                          : 'text-gray-900 dark:text-gray-100'
                      }`}
                    >
                      {day}
                    </span>
                    {count > 0 && (
                      <div className="mt-1 flex flex-wrap gap-0.5">
                        {count <= 3 ? (
                          sessions!.map((s) => (
                            <span
                              key={s.session_id}
                              className={`block h-1.5 w-1.5 rounded-full ${
                                s.session_type === 'theory'
                                  ? 'bg-blue-500'
                                  : 'bg-emerald-500'
                              }`}
                            />
                          ))
                        ) : (
                          <span className="text-xs font-medium text-indigo-600 dark:text-indigo-400">
                            {count} oturum
                          </span>
                        )}
                      </div>
                    )}
                  </button>
                );
              })}
            </div>

            {isLoading && (
              <div className="mt-4 text-center text-sm text-gray-500">
                Yükleniyor...
              </div>
            )}

            {/* Legend */}
            <div className="mt-4 flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="flex items-center gap-1">
                <span className="h-2 w-2 rounded-full bg-blue-500" /> Teorik
              </span>
              <span className="flex items-center gap-1">
                <span className="h-2 w-2 rounded-full bg-emerald-500" /> Uygulama
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Full session table for selected date */}
      {selectedDate && selectedSessions.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Oturum Detayları</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Ders Kodu</TableHead>
                  <TableHead>Ders Adı</TableHead>
                  <TableHead>Tür</TableHead>
                  <TableHead>Hafta</TableHead>
                  <TableHead>Başlangıç</TableHead>
                  <TableHead>Bitiş</TableHead>
                  <TableHead>Katılım</TableHead>
                  <TableHead>Durum</TableHead>
                  <TableHead className="text-right">İşlemler</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {selectedSessions.map((s) => {
                  const rate =
                    s.enrolled_count > 0
                      ? Math.round((s.present_count / s.enrolled_count) * 100)
                      : 0;
                  return (
                    <TableRow key={s.session_id}>
                      <TableCell className="font-medium">
                        {s.course_code}
                      </TableCell>
                      <TableCell>{s.course_name}</TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            s.session_type === 'theory' ? 'default' : 'secondary'
                          }
                        >
                          {s.session_type === 'theory' ? 'Teorik' : 'Uygulama'}
                        </Badge>
                      </TableCell>
                      <TableCell>{s.week_number}</TableCell>
                      <TableCell>
                        {new Date(s.started_at).toLocaleTimeString('tr-TR', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </TableCell>
                      <TableCell>
                        {new Date(s.expires_at).toLocaleTimeString('tr-TR', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </TableCell>
                      <TableCell>
                        <span
                          className={
                            rate >= 70
                              ? 'text-green-600 dark:text-green-400'
                              : rate >= 50
                                ? 'text-yellow-600 dark:text-yellow-400'
                                : 'text-red-600 dark:text-red-400'
                          }
                        >
                          {s.present_count}/{s.enrolled_count} (%{rate})
                        </span>
                      </TableCell>
                      <TableCell>
                        {s.is_active ? (
                          <Badge
                            variant="default"
                            className="bg-green-500"
                          >
                            Aktif
                          </Badge>
                        ) : (
                          <Badge variant="secondary">Kapandı</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button 
                          variant="outline" 
                          size="sm"
                          onClick={() => navigate(`/attendance/${s.session_id}`, { state: { session: s } })}
                        >
                          Detaylar
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
