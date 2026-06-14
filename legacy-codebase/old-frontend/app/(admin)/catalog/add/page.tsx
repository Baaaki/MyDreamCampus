'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { mockFaculties } from '@/mock_data/catalog';
import { catalogApi } from '@/lib/api-client';
import { Faculty, Department, WeeklyTopic, CourseCoordinator } from '@/lib/types';
import {
  ArrowLeft,
  Plus,
  Trash2,
  BookOpen,
  GraduationCap,
  User,
  Calendar,
  Library,
  Save,
  Loader2,
} from 'lucide-react';

// API request type
interface CreateCourseRequest {
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  class_level: number;
  credits: number;
  theoretical_hours: number;
  lab_hours: number;
  course_type: string;
  description?: string;
  prerequisites?: string[];
}

// Create course via API
const createCourse = async (data: CreateCourseRequest) => {
  return catalogApi.post('courses', { json: data }).json();
};

interface FormData {
  // Temel Bilgiler
  faculty_id: string;
  department_id: string;
  course_code: string;
  name: string;
  offering_unit: string;

  // Ders Saatleri
  semester: number;
  class_level: number;
  credits: number;
  theoretical_hours: number;
  lab_hours: number;
  ects: number;

  // Ders Özellikleri
  course_type: 'Zorunlu' | 'Seçmeli';
  education_level: string;
  teaching_type: string;
  language: string;

  // Koordinatör Bilgileri
  coordinator: CourseCoordinator;

  // İçerik
  purpose: string;
  learning_outcomes: string[];
  weekly_topics: WeeklyTopic[];
  recommended_sources: string[];

  // Açıklama
  description: string;
}

const initialFormData: FormData = {
  faculty_id: '',
  department_id: '',
  course_code: '',
  name: '',
  offering_unit: '',
  semester: 1,
  class_level: 1,
  credits: 1,
  theoretical_hours: 0,
  lab_hours: 0,
  ects: 0,
  course_type: 'Zorunlu',
  education_level: 'Lisans',
  teaching_type: 'Örgün Öğretim',
  language: 'Türkçe',
  coordinator: {
    title: '',
    name: '',
    email: '',
    phone: '',
    office: '',
  },
  purpose: '',
  learning_outcomes: [''],
  weekly_topics: [{ week: 1, topic: '', description: '' }],
  recommended_sources: [''],
  description: '',
};

export default function AddCoursePage() {
  const router = useRouter();
  const [formData, setFormData] = useState<FormData>(initialFormData);
  const [selectedFaculty, setSelectedFaculty] = useState<Faculty | null>(null);
  const [departments, setDepartments] = useState<Department[]>([]);

  // Mutation for creating course
  const createMutation = useMutation({
    mutationFn: createCourse,
    onSuccess: () => {
      alert('Ders başarıyla eklendi!');
      router.push('/catalog');
    },
    onError: async (error: any) => {
      console.error('Ders eklenirken hata:', error);
      let message = error.message;
      try {
        const body = await error.response?.json();
        if (body?.error) message = body.error;
      } catch { /* ignore parse errors */ }
      alert(`Hata: ${message}`);
    },
  });

  // Fakülte değiştiğinde bölümleri güncelle
  useEffect(() => {
    if (formData.faculty_id) {
      const faculty = mockFaculties.find(f => f.id === formData.faculty_id);
      if (faculty) {
        setSelectedFaculty(faculty);
        setDepartments(faculty.departments);
        // Bölüm seçimini sıfırla
        setFormData(prev => ({ ...prev, department_id: '' }));
      }
    } else {
      setSelectedFaculty(null);
      setDepartments([]);
    }
  }, [formData.faculty_id]);

  const handleInputChange = (field: keyof FormData, value: string | number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleCoordinatorChange = (field: keyof CourseCoordinator, value: string) => {
    setFormData(prev => ({
      ...prev,
      coordinator: { ...prev.coordinator, [field]: value },
    }));
  };

  // Öğrenme kazanımları yönetimi
  const addLearningOutcome = () => {
    setFormData(prev => ({
      ...prev,
      learning_outcomes: [...prev.learning_outcomes, ''],
    }));
  };

  const updateLearningOutcome = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      learning_outcomes: prev.learning_outcomes.map((item, i) =>
        i === index ? value : item
      ),
    }));
  };

  const removeLearningOutcome = (index: number) => {
    if (formData.learning_outcomes.length > 1) {
      setFormData(prev => ({
        ...prev,
        learning_outcomes: prev.learning_outcomes.filter((_, i) => i !== index),
      }));
    }
  };

  // Haftalık konular yönetimi
  const addWeeklyTopic = () => {
    const nextWeek = formData.weekly_topics.length + 1;
    setFormData(prev => ({
      ...prev,
      weekly_topics: [...prev.weekly_topics, { week: nextWeek, topic: '', description: '' }],
    }));
  };

  const updateWeeklyTopic = (index: number, field: keyof WeeklyTopic, value: string | number) => {
    setFormData(prev => ({
      ...prev,
      weekly_topics: prev.weekly_topics.map((item, i) =>
        i === index ? { ...item, [field]: value } : item
      ),
    }));
  };

  const removeWeeklyTopic = (index: number) => {
    if (formData.weekly_topics.length > 1) {
      setFormData(prev => ({
        ...prev,
        weekly_topics: prev.weekly_topics
          .filter((_, i) => i !== index)
          .map((item, i) => ({ ...item, week: i + 1 })),
      }));
    }
  };

  // Kaynaklar yönetimi
  const addSource = () => {
    setFormData(prev => ({
      ...prev,
      recommended_sources: [...prev.recommended_sources, ''],
    }));
  };

  const updateSource = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      recommended_sources: prev.recommended_sources.map((item, i) =>
        i === index ? value : item
      ),
    }));
  };

  const removeSource = (index: number) => {
    if (formData.recommended_sources.length > 1) {
      setFormData(prev => ({
        ...prev,
        recommended_sources: prev.recommended_sources.filter((_, i) => i !== index),
      }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Get faculty and department names from IDs
    const faculty = mockFaculties.find(f => f.id === formData.faculty_id);
    const department = departments.find(d => d.id === formData.department_id);

    if (!faculty || !department) {
      alert('Lütfen fakülte ve bölüm seçin!');
      return;
    }

    // Prepare data for API
    const requestData: CreateCourseRequest = {
      course_code: formData.course_code,
      name: formData.name,
      faculty: faculty.name,
      department: department.name,
      class_level: formData.class_level,
      credits: formData.credits,
      theoretical_hours: formData.theoretical_hours,
      lab_hours: formData.lab_hours,
      course_type: formData.course_type === 'Zorunlu' ? 'mandatory' : 'elective',
      description: formData.description || undefined,
    };

    console.log('Course Data:', requestData);

    // Send to API
    createMutation.mutate(requestData);
  };

  return (
    <div className="container mx-auto py-6 px-4 max-w-5xl">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="sm" onClick={() => router.push('/catalog')}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Geri
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Yeni Ders Ekle</h1>
          <p className="text-muted-foreground text-sm">Ders kataloğuna yeni bir ders ekleyin</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Fakülte ve Bölüm Seçimi */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GraduationCap className="h-5 w-5" />
              Fakülte ve Bölüm
            </CardTitle>
            <CardDescription>Dersin ait olduğu fakülte ve bölümü seçin</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="faculty">Fakülte *</Label>
              <Select
                value={formData.faculty_id}
                onValueChange={(value) => handleInputChange('faculty_id', value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Fakülte seçin" />
                </SelectTrigger>
                <SelectContent>
                  {mockFaculties.map((faculty) => (
                    <SelectItem key={faculty.id} value={faculty.id}>
                      {faculty.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="department">Bölüm *</Label>
              <Select
                value={formData.department_id}
                onValueChange={(value) => handleInputChange('department_id', value)}
                disabled={!formData.faculty_id}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={formData.faculty_id ? "Bölüm seçin" : "Önce fakülte seçin"} />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((dept) => (
                    <SelectItem key={dept.id} value={dept.id}>
                      {dept.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Temel Ders Bilgileri */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BookOpen className="h-5 w-5" />
              Temel Ders Bilgileri
            </CardTitle>
            <CardDescription>Dersin temel bilgilerini girin</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="course_code">Ders Kodu *</Label>
                <Input
                  id="course_code"
                  placeholder="Örn: ATA 1001"
                  value={formData.course_code}
                  onChange={(e) => handleInputChange('course_code', e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Ders Adı *</Label>
                <Input
                  id="name"
                  placeholder="Örn: Atatürk İlkeleri ve İnkılap Tarihi I"
                  value={formData.name}
                  onChange={(e) => handleInputChange('name', e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="offering_unit">Dersi Veren Birim</Label>
              <Input
                id="offering_unit"
                placeholder="Örn: Atatürk İlkeleri ve İnkılap Tarihi Bölümü"
                value={formData.offering_unit}
                onChange={(e) => handleInputChange('offering_unit', e.target.value)}
              />
            </div>

            <Separator />

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="space-y-2">
                <Label htmlFor="semester">Dönem</Label>
                <Select
                  value={formData.semester.toString()}
                  onValueChange={(value) => handleInputChange('semester', parseInt(value))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map((sem) => (
                      <SelectItem key={sem} value={sem.toString()}>
                        {sem}. Dönem
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="class_level">Sınıf</Label>
                <Select
                  value={formData.class_level.toString()}
                  onValueChange={(value) => handleInputChange('class_level', parseInt(value))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4].map((level) => (
                      <SelectItem key={level} value={level.toString()}>
                        {level}. Sınıf
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="course_type">Ders Türü</Label>
                <Select
                  value={formData.course_type}
                  onValueChange={(value) => handleInputChange('course_type', value as 'Zorunlu' | 'Seçmeli')}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Zorunlu">Zorunlu</SelectItem>
                    <SelectItem value="Seçmeli">Seçmeli</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="language">Dil</Label>
                <Select
                  value={formData.language}
                  onValueChange={(value) => handleInputChange('language', value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Türkçe">Türkçe</SelectItem>
                    <SelectItem value="İngilizce">İngilizce</SelectItem>
                    <SelectItem value="Almanca">Almanca</SelectItem>
                    <SelectItem value="Fransızca">Fransızca</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <Separator />

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="space-y-2">
                <Label htmlFor="theoretical_hours">Teorik (Saat)</Label>
                <Input
                  id="theoretical_hours"
                  type="number"
                  min="0"
                  value={formData.theoretical_hours}
                  onChange={(e) => handleInputChange('theoretical_hours', parseInt(e.target.value) || 0)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lab_hours">Lab (Saat)</Label>
                <Input
                  id="lab_hours"
                  type="number"
                  min="0"
                  value={formData.lab_hours}
                  onChange={(e) => handleInputChange('lab_hours', parseInt(e.target.value) || 0)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="credits">Kredi *</Label>
                <Input
                  id="credits"
                  type="number"
                  min="1"
                  value={formData.credits}
                  onChange={(e) => handleInputChange('credits', parseInt(e.target.value) || 0)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ects">AKTS</Label>
                <Input
                  id="ects"
                  type="number"
                  min="0"
                  value={formData.ects}
                  onChange={(e) => handleInputChange('ects', parseInt(e.target.value) || 0)}
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="education_level">Eğitim Seviyesi</Label>
                <Select
                  value={formData.education_level}
                  onValueChange={(value) => handleInputChange('education_level', value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Lisans">Lisans</SelectItem>
                    <SelectItem value="Yüksek Lisans">Yüksek Lisans</SelectItem>
                    <SelectItem value="Doktora">Doktora</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="teaching_type">Öğretim Türü</Label>
                <Select
                  value={formData.teaching_type}
                  onValueChange={(value) => handleInputChange('teaching_type', value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Örgün Öğretim">Örgün Öğretim</SelectItem>
                    <SelectItem value="İkinci Öğretim">İkinci Öğretim</SelectItem>
                    <SelectItem value="Uzaktan Öğretim">Uzaktan Öğretim</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Ders Koordinatörü */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="h-5 w-5" />
              Ders Koordinatörü
            </CardTitle>
            <CardDescription>Dersin koordinatör bilgilerini girin</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="coordinator_title">Unvan</Label>
                <Select
                  value={formData.coordinator.title}
                  onValueChange={(value) => handleCoordinatorChange('title', value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Unvan seçin" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Prof. Dr.">Prof. Dr.</SelectItem>
                    <SelectItem value="Doç. Dr.">Doç. Dr.</SelectItem>
                    <SelectItem value="Dr. Öğr. Üyesi">Dr. Öğr. Üyesi</SelectItem>
                    <SelectItem value="Öğr. Gör.">Öğr. Gör.</SelectItem>
                    <SelectItem value="Arş. Gör.">Arş. Gör.</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="coordinator_name">Ad Soyad</Label>
                <Input
                  id="coordinator_name"
                  placeholder="Örn: Mehmet Yılmaz"
                  value={formData.coordinator.name}
                  onChange={(e) => handleCoordinatorChange('name', e.target.value)}
                />
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="coordinator_email">E-posta</Label>
                <Input
                  id="coordinator_email"
                  type="email"
                  placeholder="ornek@deu.edu.tr"
                  value={formData.coordinator.email}
                  onChange={(e) => handleCoordinatorChange('email', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="coordinator_phone">Telefon</Label>
                <Input
                  id="coordinator_phone"
                  placeholder="0232 XXX XX XX"
                  value={formData.coordinator.phone}
                  onChange={(e) => handleCoordinatorChange('phone', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="coordinator_office">Ofis</Label>
                <Input
                  id="coordinator_office"
                  placeholder="Örn: A Blok, Kat 3, Oda 301"
                  value={formData.coordinator.office}
                  onChange={(e) => handleCoordinatorChange('office', e.target.value)}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Ders İçeriği */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BookOpen className="h-5 w-5" />
              Ders İçeriği
            </CardTitle>
            <CardDescription>Dersin amacı ve açıklaması</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="purpose">Dersin Amacı</Label>
              <Textarea
                id="purpose"
                placeholder="Bu dersin amacı..."
                value={formData.purpose}
                onChange={(e) => handleInputChange('purpose', e.target.value)}
                rows={3}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Ders Açıklaması</Label>
              <Textarea
                id="description"
                placeholder="Ders hakkında genel bilgi..."
                value={formData.description}
                onChange={(e) => handleInputChange('description', e.target.value)}
                rows={3}
              />
            </div>
          </CardContent>
        </Card>

        {/* Öğrenme Kazanımları */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GraduationCap className="h-5 w-5" />
              Öğrenme Kazanımları
            </CardTitle>
            <CardDescription>Bu dersi başarıyla tamamlayan öğrenci neler yapabilecek?</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {formData.learning_outcomes.map((outcome, index) => (
              <div key={index} className="flex gap-2 items-start">
                <span className="flex-shrink-0 w-8 h-9 flex items-center justify-center bg-muted rounded text-sm font-medium">
                  {index + 1}
                </span>
                <Input
                  placeholder={`${index + 1}. kazanım...`}
                  value={outcome}
                  onChange={(e) => updateLearningOutcome(index, e.target.value)}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeLearningOutcome(index)}
                  disabled={formData.learning_outcomes.length === 1}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={addLearningOutcome}>
              <Plus className="h-4 w-4 mr-2" />
              Kazanım Ekle
            </Button>
          </CardContent>
        </Card>

        {/* Haftalık Konular */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Calendar className="h-5 w-5" />
              Haftalık Konular
            </CardTitle>
            <CardDescription>14 haftalık ders planı</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {formData.weekly_topics.map((topic, index) => (
              <div key={index} className="p-4 border rounded-lg space-y-3">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-sm">Hafta {topic.week}</span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeWeeklyTopic(index)}
                    disabled={formData.weekly_topics.length === 1}
                    className="text-destructive hover:text-destructive"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
                <div className="space-y-2">
                  <Label>Konu</Label>
                  <Input
                    placeholder="Haftalık konu başlığı"
                    value={topic.topic}
                    onChange={(e) => updateWeeklyTopic(index, 'topic', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Açıklama (Opsiyonel)</Label>
                  <Textarea
                    placeholder="Konu hakkında ek açıklama..."
                    value={topic.description || ''}
                    onChange={(e) => updateWeeklyTopic(index, 'description', e.target.value)}
                    rows={2}
                  />
                </div>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={addWeeklyTopic}>
              <Plus className="h-4 w-4 mr-2" />
              Hafta Ekle
            </Button>
          </CardContent>
        </Card>

        {/* Önerilen Kaynaklar */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Library className="h-5 w-5" />
              Önerilen Kaynaklar
            </CardTitle>
            <CardDescription>Ders için önerilen kitap ve kaynaklar</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {formData.recommended_sources.map((source, index) => (
              <div key={index} className="flex gap-2 items-start">
                <span className="flex-shrink-0 w-8 h-9 flex items-center justify-center bg-muted rounded text-sm font-medium">
                  {index + 1}
                </span>
                <Input
                  placeholder="Kaynak adı, yazar, yayınevi..."
                  value={source}
                  onChange={(e) => updateSource(index, e.target.value)}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeSource(index)}
                  disabled={formData.recommended_sources.length === 1}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={addSource}>
              <Plus className="h-4 w-4 mr-2" />
              Kaynak Ekle
            </Button>
          </CardContent>
        </Card>

        {/* Submit Buttons */}
        <div className="flex justify-end gap-4 pb-8">
          <Button
            type="button"
            variant="outline"
            onClick={() => router.push('/catalog')}
            disabled={createMutation.isPending}
          >
            İptal
          </Button>
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Save className="h-4 w-4 mr-2" />
            )}
            {createMutation.isPending ? 'Kaydediliyor...' : 'Dersi Kaydet'}
          </Button>
        </div>
      </form>
    </div>
  );
}
