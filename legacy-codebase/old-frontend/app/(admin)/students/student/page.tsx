"use client";

import { useState, useEffect } from "react";
import { 
  Users, 
  Search, 
  ArrowLeft, 
  Mail, 
  Phone, 
  MapPin, 
  Calendar, 
  Book, 
  GraduationCap, 
  Award,
  Edit,
  Save,
  Printer,
  Camera,
  Trash2,
  Plus,
  User
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { studentApi } from "@/lib/api-client";
import { mockFaculties } from "@/mock_data";
import { mockStudents } from "@/mock_data/students";
import type { Student } from "@/lib/types";

// Helper to delay execution (simulate API latency)
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

export default function StudentPage() {
  // View state
  const [viewMode, setViewMode] = useState<'selection' | 'list' | 'profile'>('selection');
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

  // Selection state
  const [selectedFacultyId, setSelectedFacultyId] = useState<string>("");
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<string>("");

  // Data state
  const [studentList, setStudentList] = useState<Student[]>([]);
  const [profile, setProfile] = useState<Student | null>(null);

  // Available departments based on selected faculty
  const availableDepartments = selectedFacultyId 
    ? mockFaculties.find(f => f.id === selectedFacultyId)?.departments || []
    : [];

  // Reset logic when faculty changes
  const handleFacultyChange = (facultyId: string) => {
    setSelectedFacultyId(facultyId);
    setSelectedDepartmentId(""); // Reset department when faculty changes
  };

  // Fetch students based on selection
  const handleFetchStudents = async () => {
    if (!selectedFacultyId || !selectedDepartmentId) return;

    setIsLoading(true);
    // Simulate API call
    await delay(600);
    
    // Get faculty and department names to filter
    const faculty = mockFaculties.find(f => f.id === selectedFacultyId);
    const department = faculty?.departments.find(d => d.id === selectedDepartmentId);
    
    if (faculty && department) {
      // Filter mock students
      // Note: In a real app, we would pass IDs to the API. 
      // Here we match by name since mock data uses names for these fields currently.
      const filtered = mockStudents.filter(s => 
        (s.faculty === faculty.name || s.department === department.name)
      );
      setStudentList(filtered);
      setViewMode('list');
    }
    
    setIsLoading(false);
  };

  // Select a student to view details
  const handleStudentSelect = async (studentId: string) => {
    setIsLoading(true);
    
    try {
      // In a real app, fetch single student by ID
      // const data = await studentApi.get(`students/${studentId}`).json<Student>();
      
      // For mock, find in list
      const student = mockStudents.find(s => s.id === studentId);
      
      if (student) {
        setProfile(student);
        setViewMode('profile');
      } else {
        console.error('Student not found');
      }
    } catch (error) {
      console.error('Failed to fetch student:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Go back to list
  const handleBackToList = () => {
    setViewMode('list');
    setProfile(null);
    setIsEditing(false);
  };

  // Go back to selection
  const handleBackToSelection = () => {
    setViewMode('selection');
    setStudentList([]);
    setProfile(null);
  };

  // Handle profile changes
  const handleProfileChange = (field: keyof Student, value: any) => {
    setProfile(prev => prev ? ({ ...prev, [field]: value }) : null);
  };

  // Save changes
  const handleSave = async () => {
    if (!profile) return;

    try {
      setIsSaving(true);
      await studentApi.put(`students/${profile.id}`, { json: profile }).json();
      
      setIsEditing(false);
      alert('Öğrenci bilgileri başarıyla güncellendi!');
    } catch (error) {
      console.error('Failed to save student:', error);
      alert('Kaydetme hatası oluştu!');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50/50 pb-8">
      
      {/* ========== SELECTION VIEW ========== */}
      {viewMode === 'selection' && (
        <div className="container max-w-lg mx-auto pt-20 px-4">
          <Card className="border-t-4 border-t-[#005a87] shadow-lg">
            <CardHeader className="text-center pb-2">
              <div className="mx-auto bg-blue-50 p-3 rounded-full w-fit mb-4">
                <GraduationCap className="h-8 w-8 text-[#005a87]" />
              </div>
              <CardTitle className="text-2xl font-bold text-gray-800">Öğrenci Sorgulama</CardTitle>
              <CardDescription>
                Lütfen listelemek istediğiniz öğrencilerin fakülte ve bölümünü seçiniz.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-6">
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="faculty" className="text-gray-700 font-medium">Fakülte Seçin</Label>
                  <Select value={selectedFacultyId} onValueChange={handleFacultyChange}>
                    <SelectTrigger id="faculty" className="h-12 bg-gray-50 border-gray-200">
                      <SelectValue placeholder="Fakülte seçiniz..." />
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
                  <Label htmlFor="department" className="text-gray-700 font-medium">Bölüm Seçin</Label>
                  <Select 
                    value={selectedDepartmentId} 
                    onValueChange={setSelectedDepartmentId}
                    disabled={!selectedFacultyId}
                  >
                    <SelectTrigger id="department" className="h-12 bg-gray-50 border-gray-200">
                      <SelectValue placeholder={!selectedFacultyId ? "Önce fakülte seçiniz" : "Bölüm seçiniz..."} />
                    </SelectTrigger>
                    <SelectContent>
                      {availableDepartments.map((dept) => (
                        <SelectItem key={dept.id} value={dept.id}>
                          {dept.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <Button 
                  className="w-full h-12 text-lg font-medium bg-[#005a87] hover:bg-[#00486c] transition-all shadow-md hover:shadow-lg"
                  onClick={handleFetchStudents}
                  disabled={!selectedFacultyId || !selectedDepartmentId || isLoading}
                >
                  {isLoading ? (
                    <>
                      <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
                      Getiriliyor...
                    </>
                  ) : (
                    <>
                      <Search className="mr-2 h-5 w-5" />
                      Öğrencileri Getir
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* ========== LIST VIEW ========== */}
      {viewMode === 'list' && (
        <div className="container mx-auto px-4 py-8">
          <div className="max-w-6xl mx-auto space-y-6">
            {/* Header / Back Button */}
            <div className="flex items-center justify-between">
              <Button variant="ghost" className="text-gray-600 hover:text-gray-900 -ml-2" onClick={handleBackToSelection}>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Seçim Ekranına Dön
              </Button>
              
              <div className="text-right">
                <h2 className="text-lg font-semibold text-gray-800">
                  {mockFaculties.find(f => f.id === selectedFacultyId)?.name}
                </h2>
                <p className="text-sm text-gray-500">
                  {mockFaculties.find(f => f.id === selectedFacultyId)?.departments.find(d => d.id === selectedDepartmentId)?.name}
                </p>
              </div>
            </div>

            <Card className="border-t-4 border-t-[#005a87] shadow-md">
              <CardHeader className="border-b bg-gray-50/50">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="bg-[#005a87]/10 p-2 rounded-lg">
                      <GraduationCap className="h-5 w-5 text-[#005a87]" />
                    </div>
                    <div>
                      <CardTitle>Öğrenci Listesi</CardTitle>
                      <CardDescription>{studentList.length} öğrenci listeleniyor</CardDescription>
                    </div>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                {studentList.length === 0 ? (
                  <div className="p-12 text-center text-gray-500">
                    <Users className="h-12 w-12 mx-auto mb-4 text-gray-300" />
                    <p className="text-lg font-medium">Bu bölümde kayıtlı öğrenci bulunamadı.</p>
                  </div>
                ) : (
                  <Table>
                    <TableHeader className="bg-gray-50">
                      <TableRow>
                        <TableHead className="w-[100px]">No</TableHead>
                        <TableHead>Ad Soyad</TableHead>
                        <TableHead>E-posta</TableHead>
                        <TableHead>Sınıf</TableHead>
                        <TableHead>Danışman</TableHead>
                        <TableHead className="text-right">İşlemler</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {studentList.map((student) => (
                        <TableRow 
                          key={student.id} 
                          className="cursor-pointer hover:bg-blue-50/50 transition-colors"
                          onClick={() => handleStudentSelect(student.id)}
                        >
                          <TableCell className="font-medium">{student.student_number}</TableCell>
                          <TableCell className="font-medium text-[#005a87]">
                            {student.first_name} {student.last_name}
                          </TableCell>
                          <TableCell className="text-gray-500">{student.email}</TableCell>
                          <TableCell>
                            <span className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs font-medium">
                              {student.class_level}. Sınıf
                            </span>
                          </TableCell>
                          <TableCell className="text-gray-500">
                             {student.advisor ? `${student.advisor.first_name} ${student.advisor.last_name}` : '-'}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button 
                              size="sm" 
                              variant="outline"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleStudentSelect(student.id);
                              }}
                            >
                              <User className="h-4 w-4 mr-1" />
                              Detay
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {/* ========== PROFILE VIEW ========== */}
      {viewMode === 'profile' && profile && (
        <>
          {/* Back button */}
          <div className="bg-gray-100 dark:bg-gray-900 border-b">
            <div className="container mx-auto px-4 py-2">
              <Button variant="ghost" onClick={handleBackToList} className="text-[#005a87]">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Listeye Dön
              </Button>
            </div>
          </div>
          
          {/* Header */}
          <div className="bg-[#005a87] text-white">
            <div className="container mx-auto px-4 py-6">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-6">
                  {/* Profile Image - Placeholder */}
                  <div className="relative group">
                    <div className="w-28 h-28 rounded-lg overflow-hidden border-4 border-white/30 shadow-lg bg-white/10 flex items-center justify-center">
                      <User className="w-16 h-16 text-white/60" />
                    </div>
                  </div>
                  
                  {/* Info */}
                  <div className="flex-1">
                    <h1 className="text-2xl font-bold">
                      {profile.first_name.toUpperCase()} {profile.last_name.toUpperCase()}
                    </h1>
                    <p className="text-lg font-medium text-blue-100 mt-1">
                      {profile.student_number}
                    </p>
                    <p className="text-sm text-blue-200 mt-1">
                      {profile.department} | {profile.class_level}. Sınıf
                    </p>
                    
                    <div className="flex items-center gap-6 mt-4 text-sm">
                      <div className="flex items-center gap-2">
                        <Mail className="h-4 w-4" />
                        <span>{profile.email}</span>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* Action Buttons */}
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="bg-white/10 border-white/20 text-white hover:bg-white/20">
                    <Printer className="h-4 w-4 mr-2" />
                    Öğrenci Belgesi
                  </Button>
                  {!isEditing ? (
                    <Button size="sm" onClick={() => setIsEditing(true)} className="bg-white text-[#005a87] hover:bg-blue-50">
                      Düzenle
                    </Button>
                  ) : (
                    <Button size="sm" onClick={handleSave} disabled={isSaving} className="bg-emerald-600 hover:bg-emerald-700 border border-emerald-500 text-white">
                      <Save className="h-4 w-4 mr-2" />
                      {isSaving ? 'Kaydediliyor...' : 'Kaydet'}
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Details */}
          <div className="container mx-auto px-4 py-8">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              
              {/* Left Column: Academic Info */}
              <div className="md:col-span-2 space-y-6">
                <Card>
                  <CardHeader className="bg-blue-50/50 pb-3">
                    <CardTitle className="flex items-center gap-2 text-[#005a87]">
                      <GraduationCap className="h-5 w-5" />
                      Akademik Bilgiler
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div className="space-y-2">
                        <Label className="text-gray-500">Fakülte</Label>
                        <div className="font-medium">{profile.faculty}</div>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-gray-500">Bölüm</Label>
                        <div className="font-medium">{profile.department}</div>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-gray-500">Sınıf</Label>
                        {isEditing ? (
                           <Select 
                            value={profile.class_level.toString()} 
                            onValueChange={(val) => handleProfileChange('class_level', parseInt(val))}
                           >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="1">1. Sınıf</SelectItem>
                              <SelectItem value="2">2. Sınıf</SelectItem>
                              <SelectItem value="3">3. Sınıf</SelectItem>
                              <SelectItem value="4">4. Sınıf</SelectItem>
                            </SelectContent>
                           </Select>
                        ) : (
                          <div className="font-medium">{profile.class_level}. Sınıf</div>
                        )}
                      </div>
                      <div className="space-y-2">
                        <Label className="text-gray-500">Kayıt Yılı</Label>
                        <div className="font-medium">{profile.enrollment_year}</div>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-gray-500">Danışman</Label>
                        <div className="font-medium flex items-center gap-2">
                          <User className="h-4 w-4 text-gray-400" />
                          {profile.advisor ? `${profile.advisor.first_name} ${profile.advisor.last_name}` : 'Atanmamış'}
                        </div>
                      </div>
                      <div className="space-y-2">
                         <Label className="text-gray-500">Durum</Label>
                         <span className={`px-2 py-1 rounded text-xs font-medium inline-block ${
                           profile.status === 'active' ? 'bg-green-100 text-green-700' : 
                           profile.status === 'graduated' ? 'bg-blue-100 text-blue-700' :
                           'bg-red-100 text-red-700'
                         }`}>
                           {profile.status === 'active' ? 'Aktif Öğrenci' : 
                            profile.status === 'graduated' ? 'Mezun' : 
                            profile.status === 'suspended' ? 'Kayıt Dondurmuş' : profile.status}
                         </span>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Personal Info Edit (if editing) */}
                {isEditing && (
                  <Card>
                    <CardHeader className="bg-blue-50/50 pb-3">
                      <CardTitle className="flex items-center gap-2 text-[#005a87]">
                        <User className="h-5 w-5" />
                        Kişisel Bilgiler (Düzenleme)
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="pt-6">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                           <Label>Adı</Label>
                           <Input value={profile.first_name} onChange={(e) => handleProfileChange('first_name', e.target.value)} />
                        </div>
                        <div className="space-y-2">
                           <Label>Soyadı</Label>
                           <Input value={profile.last_name} onChange={(e) => handleProfileChange('last_name', e.target.value)} />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                           <Label>E-posta</Label>
                           <Input value={profile.email} onChange={(e) => handleProfileChange('email', e.target.value)} />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </div>

              {/* Right Column: Quick Stats / Timeline */}
              <div className="space-y-6">
                  <Card>
                    <CardHeader>
                       <CardTitle className="text-sm font-medium text-gray-500">Genel Ortalaması (GNO)</CardTitle>
                    </CardHeader>
                    <CardContent>
                       <div className="text-3xl font-bold text-[#005a87]">3.42</div>
                       <p className="text-xs text-green-600 mt-1 flex items-center">
                         <span className="w-2 h-2 rounded-full bg-green-500 mr-1"></span>
                         Başarılı
                       </p>
                    </CardContent>
                  </Card>
              </div>

            </div>
          </div>
        </>
      )}
    </div>
  );
}
