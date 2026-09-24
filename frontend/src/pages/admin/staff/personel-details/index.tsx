import { useState, useEffect } from "react"
import { staffApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import {
  adminStaffService,
  toAdminStaffRecord,
} from "@/lib/services/admin-staff-service"
import { useFaculties } from "@/lib/services/catalog-service"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Mail,
  Phone,
  Plus,
  Trash2,
  Save,
  Printer,
  GraduationCap,
  FileText,
  Award,
  Briefcase,
  BookOpen,
  User,
  Camera,
  ChevronDown,
  ArrowLeft,
  Search,
  Users,
  Building2,
  UserCog,
  ClipboardList,
  Clock,
  MapPin,
  Calendar,
} from "lucide-react"
import type { AdminStaffProfile, AdminStaffRecord, Staff } from "@/lib/types"

// Eğitim bilgisi interface
interface EducationInfo {
  id: string
  degree: string
  institution: string
  department: string
  year: number
}

// Makale interface
interface Article {
  id: string
  title: string
  journal: string
  year: number
  authors: string
  doi?: string
  journalType?: string // SCI, SCI-Expanded, SSCI, etc.
  domesticInternational?: string // Yurtiçi/Yurtdışı
  publishingMonth?: string
  issuePageYear?: string // Volume/Issue/Page info
  language?: string
  articleType?: string // Research article, Review, etc.
}

// Bildiri interface
interface Bulletin {
  id: string
  title: string
  conference: string
  year: number
  location: string
}

// Proje interface
interface Project {
  id: string
  title: string
  role: string
  funder: string
  startYear: number
  endYear?: number
  status: "ongoing" | "completed"
}

// Ödül interface
interface AwardItem {
  id: string
  title: string
  institution: string
  year: number
}

// Burs interface
interface Scholarship {
  id: string
  title: string
  institution: string
  year: number
}

// İdari görev interface
interface AdminAssignment {
  id: string
  title: string
  institution: string
  startYear: number
  endYear?: number
}

// Profile data interface
interface StaffProfile {
  id?: string
  title: string
  firstName: string
  lastName: string
  faculty: string
  department: string
  email: string
  phone: string
  profileImage?: string
  education: EducationInfo[]
  articles: Article[]
  bulletins: Bulletin[]
  projects: Project[]
  awards: AwardItem[]
  scholarships: Scholarship[]
  adminAssignments: AdminAssignment[]
}

// API response interface (snake_case from backend)
interface ApiTeacherProfileResponse {
  id: string
  staff_id: string
  academic_title?: string
  faculty?: string
  first_name: string
  last_name: string
  department: string
  email: string
  phone?: string
  office_location?: string
  profile_image_url?: string
  education: Array<{
    id: string
    degree: string
    institution: string
    department: string
    year: number
  }>
  articles: Array<{
    id: string
    title: string
    journal: string
    year: number
    authors: string
    doi?: string
    journalType?: string
    domesticInternational?: string
    language?: string
    articleType?: string
  }>
  bulletins: Array<{
    id: string
    title: string
    conference: string
    year: number
    location: string
  }>
  projects: Array<{
    id: string
    title: string
    role: string
    funder: string
    startYear: number
    endYear?: number
    status: "ongoing" | "completed"
  }>
  awards: Array<{
    id: string
    title: string
    institution: string
    year: number
  }>
  scholarships: Array<{
    id: string
    title: string
    institution: string
    year: number
  }>
  admin_assignments: Array<{
    id: string
    title: string
    institution: string
    startYear: number
    endYear?: number
  }>
}

// Transform API response to frontend format
function transformApiResponseToProfile(
  data: ApiTeacherProfileResponse
): StaffProfile {
  return {
    id: data.id,
    title: data.academic_title || "",
    firstName: data.first_name,
    lastName: data.last_name,
    faculty: data.faculty || "",
    department: data.department,
    email: data.email,
    phone: data.phone || "",
    profileImage: data.profile_image_url,
    education: data.education || [],
    articles: data.articles || [],
    bulletins: data.bulletins || [],
    projects: data.projects || [],
    awards: data.awards || [],
    scholarships: data.scholarships || [],
    adminAssignments: data.admin_assignments || [],
  }
}

interface CreateAdminStaffFormData {
  email: string
  firstName: string
  lastName: string
  title: string
  position: string
  faculty: string
  department: string
  phone: string
  profileImage: string
  jobDescription: string
  responsibilities: string[]
  workingHours: string
  officeLocation: string
  startDate: string
}

const initialCreateFormData: CreateAdminStaffFormData = {
  email: "",
  firstName: "",
  lastName: "",
  title: "",
  position: "",
  faculty: "",
  department: "",
  phone: "",
  profileImage: "",
  jobDescription: "",
  responsibilities: [],
  workingHours: "",
  officeLocation: "",
  startDate: "",
}

export default function StaffProfilePage() {
  // Check if user is authenticated (has valid token)
  const [isAuthenticated, setIsAuthenticated] = useState(false)

  useEffect(() => {
    // Check for user in localStorage
    const checkAuth = () => {
      const userStr = localStorage.getItem("user")
      setIsAuthenticated(!!userStr)
    }
    checkAuth()
  }, [])

  // Staff type: 'academic' or 'administrative'
  const [staffType, setStaffType] = useState<"academic" | "administrative">(
    "academic"
  )

  // View mode: 'selection' -> 'list' -> 'profile'
  const [viewMode, setViewMode] = useState<"selection" | "list" | "profile">(
    "selection"
  )

  // Faculty & Department selection
  const [selectedFacultyId, setSelectedFacultyId] = useState<string>("")
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<string>("")
  const [staffList, setStaffList] = useState<Staff[]>([])
  const [adminStaffList, setAdminStaffList] = useState<AdminStaffRecord[]>([])
  const [selectedStaffId, setSelectedStaffId] = useState<string | null>(null)
  const [isLoadingStaff, setIsLoadingStaff] = useState(false)

  // Create admin staff dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [isCreateSubmitting, setIsCreateSubmitting] = useState(false)
  const [createError, setCreateError] = useState("")
  const [createFormData, setCreateFormData] =
    useState<CreateAdminStaffFormData>(initialCreateFormData)

  // Profile states
  const [profile, setProfile] = useState<StaffProfile | null>(null)
  const [adminProfile, setAdminProfile] = useState<AdminStaffProfile | null>(
    null
  )
  const [activeTab, setActiveTab] = useState("education")
  const [isEditing, setIsEditing] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [expandedArticles, setExpandedArticles] = useState<string[]>([])

  const {
    data: faculties = [],
    isLoading: isLoadingFaculties,
    isError: isErrorFaculties,
  } = useFaculties()

  // Get departments for selected faculty
  const selectedFaculty = faculties.find((f) => f.id === selectedFacultyId)
  const departments = selectedFaculty?.departments || []

  // Handle staff type change - reset selections
  const handleStaffTypeChange = (type: "academic" | "administrative") => {
    setStaffType(type)
    setSelectedFacultyId("")
    setSelectedDepartmentId("")
    setStaffList([])
    setAdminStaffList([])
    setViewMode("selection")
  }

  // Handle faculty change - reset department
  const handleFacultyChange = (facultyId: string) => {
    setSelectedFacultyId(facultyId)
    setSelectedDepartmentId("")
  }

  // Fetch staff list for selected faculty/department
  const handleFetchStaff = async () => {
    if (!selectedFacultyId) return
    if (staffType === "academic" && !selectedDepartmentId) return

    setIsLoadingStaff(true)

    const selectedFac = faculties.find((f) => f.id === selectedFacultyId)
    const selectedDept = departments.find((d) => d.id === selectedDepartmentId)

    if (staffType === "academic") {
      try {
        // Use real API to fetch staff, optionally filtered by department
        const departmentName = selectedDept?.name || ""
        const queryParams = departmentName
          ? `?department=${encodeURIComponent(departmentName)}`
          : ""

        console.log(
          "[Personel Details] Fetching staff with department:",
          departmentName
        )

        interface StaffListResponse {
          data: Staff[]
          pagination: {
            page: number
            limit: number
            total: number
            total_pages: number
          }
        }

        const response = (await staffApi
          .get(`instructors${queryParams}`)
          .json()) as StaffListResponse
        console.log("[Personel Details] Staff response:", response)

        setStaffList(response.data || [])
        setViewMode("list")
      } catch (error) {
        console.error("[Personel Details] Failed to fetch staff:", error)
        setStaffList([])
      } finally {
        setIsLoadingStaff(false)
      }
    } else {
      try {
        setAdminStaffList(await adminStaffService.list(selectedFac?.name ?? ""))
        setViewMode("list")
      } catch (error) {
        console.error("[Personel Details] Failed to fetch admin staff:", error)
        setAdminStaffList([])
        alert(apiErrorMessage(error, "İdari personel listesi alınamadı"))
      } finally {
        setIsLoadingStaff(false)
      }
    }
  }

  // Handle academic staff selection - load profile by ID
  const handleStaffSelect = async (staffId: string) => {
    setSelectedStaffId(staffId)
    setIsLoading(true)

    try {
      const apiData = (await staffApi
        .get(`profile/${staffId}`)
        .json()) as ApiTeacherProfileResponse
      console.log("[handleStaffSelect] Raw API response:", apiData)

      if (apiData) {
        // Transform from snake_case backend response to camelCase frontend format
        const profileData = transformApiResponseToProfile(apiData)
        console.log("[handleStaffSelect] Transformed profile:", profileData)

        setProfile(profileData)
        setAdminProfile(null)
        setViewMode("profile")
      } else {
        console.error("Profile not found for staff ID:", staffId)
      }
    } catch (error) {
      console.error("Failed to fetch profile:", error)
    } finally {
      setIsLoading(false)
    }
  }

  // Handle administrative staff selection - load profile by ID
  const handleAdminStaffSelect = async (staffId: string) => {
    setSelectedStaffId(staffId)
    setIsLoading(true)

    try {
      setAdminProfile(await adminStaffService.get(staffId))
      setProfile(null)
      setViewMode("profile")
    } catch (error) {
      console.error("Failed to fetch admin profile:", error)
      alert(apiErrorMessage(error, "İdari personel profili alınamadı"))
    } finally {
      setIsLoading(false)
    }
  }

  // Go back to list
  const handleBackToList = () => {
    setViewMode("list")
    setProfile(null)
    setAdminProfile(null)
    setIsEditing(false)
  }

  // Go back to selection
  const handleBackToSelection = () => {
    setViewMode("selection")
    setStaffList([])
    setAdminStaffList([])
  }

  // Basic info handlers
  const handleBasicInfoChange = (field: string, value: string) => {
    setProfile((prev) => (prev ? { ...prev, [field]: value } : null))
  }

  // Education handlers
  const addEducation = () => {
    const newEdu: EducationInfo = {
      id: Date.now().toString(),
      degree: "",
      institution: "",
      department: "",
      year: new Date().getFullYear(),
    }
    setProfile((prev) =>
      prev ? { ...prev, education: [...prev.education, newEdu] } : null
    )
  }

  const updateEducation = (
    id: string,
    field: keyof EducationInfo,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            education: prev.education.map((edu) =>
              edu.id === id ? { ...edu, [field]: value } : edu
            ),
          }
        : null
    )
  }

  const removeEducation = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            education: prev.education.filter((edu) => edu.id !== id),
          }
        : null
    )
  }

  // Article handlers
  const addArticle = () => {
    const newArticle: Article = {
      id: Date.now().toString(),
      title: "",
      journal: "",
      year: new Date().getFullYear(),
      authors: "",
    }
    setProfile((prev) =>
      prev ? { ...prev, articles: [...prev.articles, newArticle] } : null
    )
  }

  const updateArticle = (
    id: string,
    field: keyof Article,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            articles: prev.articles.map((art) =>
              art.id === id ? { ...art, [field]: value } : art
            ),
          }
        : null
    )
  }

  const removeArticle = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            articles: prev.articles.filter((art) => art.id !== id),
          }
        : null
    )
  }

  // Bulletin handlers
  const addBulletin = () => {
    const newBulletin: Bulletin = {
      id: Date.now().toString(),
      title: "",
      conference: "",
      year: new Date().getFullYear(),
      location: "",
    }
    setProfile((prev) =>
      prev ? { ...prev, bulletins: [...prev.bulletins, newBulletin] } : null
    )
  }

  const updateBulletin = (
    id: string,
    field: keyof Bulletin,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            bulletins: prev.bulletins.map((bul) =>
              bul.id === id ? { ...bul, [field]: value } : bul
            ),
          }
        : null
    )
  }

  const removeBulletin = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            bulletins: prev.bulletins.filter((bul) => bul.id !== id),
          }
        : null
    )
  }

  // Project handlers
  const addProject = () => {
    const newProject: Project = {
      id: Date.now().toString(),
      title: "",
      role: "",
      funder: "",
      startYear: new Date().getFullYear(),
      status: "ongoing",
    }
    setProfile((prev) =>
      prev ? { ...prev, projects: [...prev.projects, newProject] } : null
    )
  }

  const updateProject = (
    id: string,
    field: keyof Project,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            projects: prev.projects.map((proj) =>
              proj.id === id ? { ...proj, [field]: value } : proj
            ),
          }
        : null
    )
  }

  const removeProject = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            projects: prev.projects.filter((proj) => proj.id !== id),
          }
        : null
    )
  }

  // Award handlers
  const addAward = () => {
    const newAward: AwardItem = {
      id: Date.now().toString(),
      title: "",
      institution: "",
      year: new Date().getFullYear(),
    }
    setProfile((prev) =>
      prev ? { ...prev, awards: [...prev.awards, newAward] } : null
    )
  }

  const updateAward = (
    id: string,
    field: keyof AwardItem,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            awards: prev.awards.map((aw) =>
              aw.id === id ? { ...aw, [field]: value } : aw
            ),
          }
        : null
    )
  }

  const removeAward = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            awards: prev.awards.filter((aw) => aw.id !== id),
          }
        : null
    )
  }

  // Scholarship handlers
  const addScholarship = () => {
    const newScholarship: Scholarship = {
      id: Date.now().toString(),
      title: "",
      institution: "",
      year: new Date().getFullYear(),
    }
    setProfile((prev) =>
      prev
        ? { ...prev, scholarships: [...prev.scholarships, newScholarship] }
        : null
    )
  }

  const updateScholarship = (
    id: string,
    field: keyof Scholarship,
    value: string | number
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            scholarships: prev.scholarships.map((sch) =>
              sch.id === id ? { ...sch, [field]: value } : sch
            ),
          }
        : null
    )
  }

  const removeScholarship = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            scholarships: prev.scholarships.filter((sch) => sch.id !== id),
          }
        : null
    )
  }

  // Admin assignment handlers
  const addAdminAssignment = () => {
    const newAssignment: AdminAssignment = {
      id: Date.now().toString(),
      title: "",
      institution: "",
      startYear: new Date().getFullYear(),
    }
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            adminAssignments: [...prev.adminAssignments, newAssignment],
          }
        : null
    )
  }

  const updateAdminAssignment = (
    id: string,
    field: keyof AdminAssignment,
    value: string | number | undefined
  ) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            adminAssignments: prev.adminAssignments.map((assign) =>
              assign.id === id ? { ...assign, [field]: value } : assign
            ),
          }
        : null
    )
  }

  const removeAdminAssignment = (id: string) => {
    setProfile((prev) =>
      prev
        ? {
            ...prev,
            adminAssignments: prev.adminAssignments.filter(
              (assign) => assign.id !== id
            ),
          }
        : null
    )
  }

  // Admin Info Handlers (Administrative Staff)
  const handleAdminInfoChange = <K extends keyof AdminStaffProfile>(
    field: K,
    value: AdminStaffProfile[K]
  ) => {
    setAdminProfile((prev) => (prev ? { ...prev, [field]: value } : null))
  }

  const handleAdminResponsibilityChange = (index: number, value: string) => {
    setAdminProfile((prev) => {
      if (!prev) return null
      const newResponsibilities = [...prev.responsibilities]
      newResponsibilities[index] = value
      return { ...prev, responsibilities: newResponsibilities }
    })
  }

  const addAdminResponsibility = () => {
    setAdminProfile((prev) =>
      prev
        ? {
            ...prev,
            responsibilities: [...prev.responsibilities, ""],
          }
        : null
    )
  }

  const removeAdminResponsibility = (index: number) => {
    setAdminProfile((prev) => {
      if (!prev) return null
      return {
        ...prev,
        responsibilities: prev.responsibilities.filter((_, i) => i !== index),
      }
    })
  }

  // Transform frontend profile to backend API format (camelCase -> snake_case)
  const transformProfileToApiFormat = (p: StaffProfile) => ({
    academic_title: p.title,
    faculty: p.faculty,
    profile_image_url: p.profileImage,
    education: p.education,
    articles: p.articles,
    bulletins: p.bulletins,
    projects: p.projects,
    awards: p.awards,
    scholarships: p.scholarships,
    admin_assignments: p.adminAssignments,
  })

  const handleSave = async () => {
    if (staffType === "academic" && profile) {
      try {
        setIsSaving(true)
        const apiData = transformProfileToApiFormat(profile)
        const response = (await staffApi
          .put(`${selectedStaffId}/profile`, { json: apiData })
          .json()) as ApiTeacherProfileResponse

        // Transform response back to frontend format
        const updatedProfile = transformApiResponseToProfile(response)
        setProfile(updatedProfile)
        setIsEditing(false)
        alert("Profil başarıyla kaydedildi!")
      } catch (error) {
        console.error("Failed to save profile:", error)
        alert("Profil kaydedilirken bir hata oluştu!")
      } finally {
        setIsSaving(false)
      }
    } else if (staffType === "administrative" && adminProfile) {
      try {
        setIsSaving(true)
        setAdminProfile(
          await adminStaffService.update(adminProfile.id, adminProfile)
        )
        setIsEditing(false)
        alert("İdari personel profili başarıyla kaydedildi!")
      } catch (error) {
        console.error("Failed to save admin profile:", error)
        // The backend's Turkish text says which field it refused (a taken
        // e-mail, an invalid image URL); a generic message hides that.
        alert(apiErrorMessage(error, "Profil kaydedilirken bir hata oluştu!"))
      } finally {
        setIsSaving(false)
      }
    }
  }

  const handleOpenCreateDialog = () => {
    setCreateFormData({
      ...initialCreateFormData,
      faculty: selectedFaculty?.name || "",
    })
    setCreateError("")
    setCreateDialogOpen(true)
  }

  const handleCreateAdminStaff = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateError("")

    // 1. Zorunlu alanlar
    if (
      !createFormData.email.trim() ||
      !createFormData.firstName.trim() ||
      !createFormData.lastName.trim() ||
      !createFormData.faculty.trim() ||
      !createFormData.position.trim()
    ) {
      setCreateError("Lütfen tüm zorunlu alanları (*) doldurunuz.")
      return
    }

    // 2. E-posta doğrulaması
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(createFormData.email.trim())) {
      setCreateError("Lütfen geçerli bir e-posta adresi giriniz.")
      return
    }

    // 3. Tarih doğrulaması
    if (createFormData.startDate.trim()) {
      const dateRegex = /^\d{4}-\d{2}-\d{2}$/
      if (!dateRegex.test(createFormData.startDate.trim())) {
        setCreateError("Tarih formatı YYYY-AA-GG şeklinde olmalıdır.")
        return
      }
      const parsedDate = new Date(createFormData.startDate.trim())
      if (isNaN(parsedDate.getTime())) {
        setCreateError("Lütfen geçerli bir tarih giriniz.")
        return
      }
    }

    // 4. İsteğe bağlı alan kontrolleri
    if (createFormData.phone.trim().length > 20) {
      setCreateError("Telefon numarası en fazla 20 karakter olabilir.")
      return
    }

    if (
      createFormData.profileImage.trim() &&
      !/^https?:\/\//i.test(createFormData.profileImage.trim())
    ) {
      setCreateError(
        "Profil resmi URL'si yalnızca http:// veya https:// ile başlamalıdır."
      )
      return
    }

    if (createFormData.jobDescription.trim().length > 2000) {
      setCreateError("Görev tanımı en fazla 2000 karakter olabilir.")
      return
    }

    const cleanedResponsibilities = createFormData.responsibilities
      .map((r) => r.trim())
      .filter((r) => r !== "")

    if (cleanedResponsibilities.length > 30) {
      setCreateError("En fazla 30 sorumluluk maddesi girilebilir.")
      return
    }

    if (cleanedResponsibilities.some((r) => r.length > 300)) {
      setCreateError(
        "Her bir sorumluluk maddesi en fazla 300 karakter olabilir."
      )
      return
    }

    setIsCreateSubmitting(true)
    try {
      const payload: Omit<AdminStaffProfile, "id"> = {
        email: createFormData.email.trim(),
        title: createFormData.title.trim(),
        firstName: createFormData.firstName.trim(),
        lastName: createFormData.lastName.trim(),
        faculty: createFormData.faculty.trim() || selectedFaculty?.name || "",
        department: createFormData.department.trim() || undefined,
        phone: createFormData.phone.trim(),
        profileImage: createFormData.profileImage.trim() || undefined,
        position: createFormData.position.trim(),
        jobDescription: createFormData.jobDescription.trim(),
        responsibilities: cleanedResponsibilities,
        workingHours: createFormData.workingHours.trim(),
        officeLocation: createFormData.officeLocation.trim(),
        startDate: createFormData.startDate.trim(),
      }

      const created = await adminStaffService.create(payload)
      const newRecord = toAdminStaffRecord(created)

      setAdminStaffList((prev) => [...prev, newRecord])
      setCreateDialogOpen(false)
      setCreateFormData(initialCreateFormData)
    } catch (err: unknown) {
      console.error("Failed to create admin staff:", err)
      const errorMsg = apiErrorMessage(
        err,
        "İdari personel kaydı oluşturulurken bir hata oluştu"
      )
      setCreateError(errorMsg)
    } finally {
      setIsCreateSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      {/* ========== SELECTION VIEW ========== */}
      {viewMode === "selection" && (
        <div className="container mx-auto px-4 py-8">
          <Card className="mx-auto max-w-2xl">
            <CardHeader className="rounded-t-lg bg-[#005a87] text-white">
              <CardTitle className="flex items-center gap-2">
                <Building2 className="h-6 w-6" />
                Personel Detay Görüntüleme
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6 p-6">
              {/* Staff Type Selection Tabs */}
              <div className="flex border-b border-gray-200 dark:border-gray-700">
                <button
                  onClick={() => handleStaffTypeChange("academic")}
                  className={`flex-1 px-4 py-3 text-center font-medium transition-colors ${
                    staffType === "academic"
                      ? "border-b-2 border-[#005a87] bg-blue-50 text-[#005a87] dark:bg-blue-900/20"
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-700 dark:hover:bg-gray-800"
                  }`}
                >
                  <div className="flex items-center justify-center gap-2">
                    <GraduationCap className="h-5 w-5" />
                    Akademik Personel
                  </div>
                </button>
                <button
                  onClick={() => handleStaffTypeChange("administrative")}
                  className={`flex-1 px-4 py-3 text-center font-medium transition-colors ${
                    staffType === "administrative"
                      ? "border-b-2 border-[#005a87] bg-blue-50 text-[#005a87] dark:bg-blue-900/20"
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-700 dark:hover:bg-gray-800"
                  }`}
                >
                  <div className="flex items-center justify-center gap-2">
                    <UserCog className="h-5 w-5" />
                    İdari Personel
                  </div>
                </button>
              </div>

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="faculty">Fakülte Seçin</Label>
                  <Select
                    value={selectedFacultyId}
                    onValueChange={handleFacultyChange}
                    disabled={isLoadingFaculties}
                  >
                    <SelectTrigger id="faculty">
                      <SelectValue
                        placeholder={
                          isLoadingFaculties
                            ? "Fakülteler yükleniyor..."
                            : isErrorFaculties
                              ? "Fakülteler yüklenemedi"
                              : "Fakülte seçin..."
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {faculties.map((faculty) => (
                        <SelectItem key={faculty.id} value={faculty.id}>
                          {faculty.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {isErrorFaculties && (
                    <p className="text-xs text-destructive">
                      Fakülteler yüklenirken bir hata oluştu
                    </p>
                  )}
                </div>

                {/* Department selection - only for academic staff */}
                {staffType === "academic" && (
                  <div className="space-y-2">
                    <Label htmlFor="department">Bölüm Seçin</Label>
                    <Select
                      value={selectedDepartmentId}
                      onValueChange={setSelectedDepartmentId}
                      disabled={!selectedFacultyId}
                    >
                      <SelectTrigger id="department">
                        <SelectValue
                          placeholder={
                            selectedFacultyId
                              ? "Bölüm seçin..."
                              : "Önce fakülte seçin..."
                          }
                        />
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
                )}

                {staffType === "administrative" && selectedFacultyId && (
                  <p className="text-sm text-gray-500 italic">
                    İdari personel için bölüm seçimi gerekmez. Seçilen
                    fakültedeki tüm idari personel listelenecektir.
                  </p>
                )}
              </div>

              <Button
                onClick={handleFetchStaff}
                disabled={
                  !selectedFacultyId ||
                  (staffType === "academic" && !selectedDepartmentId) ||
                  isLoadingStaff
                }
                className="w-full bg-[#005a87] hover:bg-[#004a6d]"
              >
                {isLoadingStaff ? (
                  <>
                    <div className="mr-2 h-4 w-4 animate-spin rounded-full border-b-2 border-white"></div>
                    Yükleniyor...
                  </>
                ) : (
                  <>
                    <Search className="mr-2 h-4 w-4" />
                    {staffType === "academic"
                      ? "Akademik Personeli Getir"
                      : "İdari Personeli Getir"}
                  </>
                )}
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

      {/* ========== LIST VIEW ========== */}
      {viewMode === "list" && (
        <div className="container mx-auto px-4 py-8">
          <Card>
            <CardHeader className="rounded-t-lg bg-[#005a87] text-white">
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  {staffType === "academic" ? (
                    <GraduationCap className="h-6 w-6" />
                  ) : (
                    <UserCog className="h-6 w-6" />
                  )}
                  {selectedFaculty?.name} -{" "}
                  {staffType === "academic"
                    ? "Akademik Personel"
                    : "İdari Personel"}
                  {staffType === "academic" &&
                    selectedDepartmentId &&
                    departments.find((d) => d.id === selectedDepartmentId) && (
                      <span className="text-sm font-normal opacity-80">
                        (
                        {
                          departments.find((d) => d.id === selectedDepartmentId)
                            ?.name
                        }
                        )
                      </span>
                    )}
                </CardTitle>
                <div className="flex items-center gap-2">
                  {staffType === "administrative" && (
                    <Dialog
                      open={createDialogOpen}
                      onOpenChange={(open) => {
                        setCreateDialogOpen(open)
                        if (open) {
                          setCreateFormData({
                            ...initialCreateFormData,
                            faculty: selectedFaculty?.name || "",
                          })
                          setCreateError("")
                        } else {
                          setCreateError("")
                        }
                      }}
                    >
                      <DialogTrigger asChild>
                        <Button className="bg-white text-[#005a87] hover:bg-white/90">
                          <Plus className="mr-2 h-4 w-4" />
                          Yeni İdari Personel
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-[650px]">
                        <DialogHeader>
                          <DialogTitle>Yeni İdari Personel Ekle</DialogTitle>
                        </DialogHeader>

                        {createError && (
                          <div className="rounded-lg border border-destructive bg-destructive/10 px-4 py-3 text-sm text-destructive">
                            {createError}
                          </div>
                        )}

                        <form
                          onSubmit={handleCreateAdminStaff}
                          className="space-y-4"
                        >
                          {/* Fakülte (dolu gelir) */}
                          <div className="space-y-2">
                            <Label htmlFor="create_faculty">Fakülte *</Label>
                            <Input
                              id="create_faculty"
                              value={createFormData.faculty}
                              readOnly
                              disabled
                              className="cursor-not-allowed bg-muted"
                            />
                          </div>

                          {/* Ad & Soyad */}
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="create_first_name">Ad *</Label>
                              <Input
                                id="create_first_name"
                                required
                                value={createFormData.firstName}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    firstName: e.target.value,
                                  })
                                }
                                placeholder="Ad"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="create_last_name">Soyad *</Label>
                              <Input
                                id="create_last_name"
                                required
                                value={createFormData.lastName}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    lastName: e.target.value,
                                  })
                                }
                                placeholder="Soyad"
                              />
                            </div>
                          </div>

                          {/* E-posta */}
                          <div className="space-y-2">
                            <Label htmlFor="create_email">E-posta *</Label>
                            <Input
                              id="create_email"
                              type="email"
                              required
                              value={createFormData.email}
                              onChange={(e) =>
                                setCreateFormData({
                                  ...createFormData,
                                  email: e.target.value,
                                })
                              }
                              placeholder="ornek@uni.edu.tr"
                            />
                          </div>

                          {/* Pozisyon & Unvan */}
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="create_position">
                                Pozisyon *
                              </Label>
                              <Input
                                id="create_position"
                                required
                                value={createFormData.position}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    position: e.target.value,
                                  })
                                }
                                placeholder="Fakülte Sekreteri, Memur vb."
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="create_title">Unvan</Label>
                              <Input
                                id="create_title"
                                value={createFormData.title}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    title: e.target.value,
                                  })
                                }
                                placeholder="Şef, Uzman vb."
                              />
                            </div>
                          </div>

                          {/* Bölüm & Telefon */}
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="create_department">Bölüm</Label>
                              {departments.length > 0 ? (
                                <Select
                                  value={
                                    createFormData.department || "__none__"
                                  }
                                  onValueChange={(val) =>
                                    setCreateFormData({
                                      ...createFormData,
                                      department: val === "__none__" ? "" : val,
                                    })
                                  }
                                >
                                  <SelectTrigger
                                    id="create_department"
                                    className="w-full"
                                  >
                                    <SelectValue placeholder="Bölüm seçin (opsiyonel)" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="__none__">
                                      Genel / Bölüm Yok
                                    </SelectItem>
                                    {departments.map((dept) => (
                                      <SelectItem
                                        key={dept.id}
                                        value={dept.name}
                                      >
                                        {dept.name}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              ) : (
                                <Input
                                  id="create_department"
                                  value={createFormData.department}
                                  onChange={(e) =>
                                    setCreateFormData({
                                      ...createFormData,
                                      department: e.target.value,
                                    })
                                  }
                                  placeholder="Bölüm adı"
                                />
                              )}
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="create_phone">Telefon</Label>
                              <Input
                                id="create_phone"
                                type="tel"
                                maxLength={20}
                                value={createFormData.phone}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    phone: e.target.value,
                                  })
                                }
                                placeholder="+90 232 301 1001"
                              />
                            </div>
                          </div>

                          {/* Ofis Konumu & Çalışma Saatleri */}
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="create_office_location">
                                Ofis Konumu
                              </Label>
                              <Input
                                id="create_office_location"
                                maxLength={200}
                                value={createFormData.officeLocation}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    officeLocation: e.target.value,
                                  })
                                }
                                placeholder="Dekanlık, 101"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="create_working_hours">
                                Çalışma Saatleri
                              </Label>
                              <Input
                                id="create_working_hours"
                                maxLength={100}
                                value={createFormData.workingHours}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    workingHours: e.target.value,
                                  })
                                }
                                placeholder="08:30 - 17:30"
                              />
                            </div>
                          </div>

                          {/* Göreve Başlama Tarihi & Profil Resmi URL */}
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="create_start_date">
                                Göreve Başlama Tarihi
                              </Label>
                              <Input
                                id="create_start_date"
                                type="date"
                                value={createFormData.startDate}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    startDate: e.target.value,
                                  })
                                }
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="create_profile_image">
                                Profil Resmi URL
                              </Label>
                              <Input
                                id="create_profile_image"
                                type="url"
                                maxLength={2048}
                                value={createFormData.profileImage}
                                onChange={(e) =>
                                  setCreateFormData({
                                    ...createFormData,
                                    profileImage: e.target.value,
                                  })
                                }
                                placeholder="https://..."
                              />
                            </div>
                          </div>

                          {/* Görev Tanımı */}
                          <div className="space-y-2">
                            <Label htmlFor="create_job_description">
                              Görev Tanımı
                            </Label>
                            <Textarea
                              id="create_job_description"
                              maxLength={2000}
                              value={createFormData.jobDescription}
                              onChange={(e) =>
                                setCreateFormData({
                                  ...createFormData,
                                  jobDescription: e.target.value,
                                })
                              }
                              placeholder="İdari personelin görev ve sorumluluk alanı..."
                              rows={3}
                            />
                          </div>

                          {/* Sorumluluklar */}
                          <div className="space-y-2">
                            <div className="flex items-center justify-between">
                              <Label>Sorumluluklar</Label>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() =>
                                  setCreateFormData({
                                    ...createFormData,
                                    responsibilities: [
                                      ...createFormData.responsibilities,
                                      "",
                                    ],
                                  })
                                }
                                disabled={
                                  createFormData.responsibilities.length >= 30
                                }
                              >
                                <Plus className="mr-1 h-3.5 w-3.5" />
                                Madde Ekle
                              </Button>
                            </div>
                            {createFormData.responsibilities.length > 0 && (
                              <div className="max-h-40 space-y-2 overflow-y-auto pr-1">
                                {createFormData.responsibilities.map(
                                  (resp, idx) => (
                                    <div
                                      key={idx}
                                      className="flex items-center gap-2"
                                    >
                                      <Input
                                        value={resp}
                                        maxLength={300}
                                        onChange={(e) => {
                                          const updated = [
                                            ...createFormData.responsibilities,
                                          ]
                                          updated[idx] = e.target.value
                                          setCreateFormData({
                                            ...createFormData,
                                            responsibilities: updated,
                                          })
                                        }}
                                        placeholder={`Sorumluluk maddesi ${idx + 1}`}
                                      />
                                      <Button
                                        type="button"
                                        variant="ghost"
                                        size="sm"
                                        className="h-9 w-9 p-0 text-red-500 hover:text-red-700"
                                        onClick={() => {
                                          setCreateFormData({
                                            ...createFormData,
                                            responsibilities:
                                              createFormData.responsibilities.filter(
                                                (_, i) => i !== idx
                                              ),
                                          })
                                        }}
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </div>
                                  )
                                )}
                              </div>
                            )}
                          </div>

                          {/* Form Butonları */}
                          <div className="flex gap-2 pt-4">
                            <Button
                              type="submit"
                              disabled={isCreateSubmitting}
                              className="flex-1"
                            >
                              {isCreateSubmitting
                                ? "Oluşturuluyor..."
                                : "Personel Oluştur"}
                            </Button>
                            <Button
                              type="button"
                              variant="outline"
                              onClick={() => {
                                setCreateDialogOpen(false)
                                setCreateError("")
                              }}
                              disabled={isCreateSubmitting}
                              className="flex-1"
                            >
                              İptal
                            </Button>
                          </div>
                        </form>
                      </DialogContent>
                    </Dialog>
                  )}
                  <Button
                    variant="ghost"
                    onClick={handleBackToSelection}
                    className="text-white hover:bg-white/20"
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Geri
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {/* Academic Staff List */}
              {staffType === "academic" && (
                <>
                  {staffList.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">
                      <Users className="mx-auto mb-4 h-12 w-12 opacity-50" />
                      <p>Bu bölümde kayıtlı akademik personel bulunamadı.</p>
                      <Button
                        variant="outline"
                        onClick={handleBackToSelection}
                        className="mt-4"
                      >
                        Farklı Bölüm Seç
                      </Button>
                    </div>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-gray-50 dark:bg-gray-800">
                          <TableHead className="font-semibold">
                            Ad Soyad
                          </TableHead>
                          <TableHead className="font-semibold">
                            E-posta
                          </TableHead>
                          <TableHead className="font-semibold">Ofis</TableHead>
                          <TableHead className="font-semibold">Durum</TableHead>
                          <TableHead className="text-right font-semibold">
                            İşlem
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {staffList.map((staff) => (
                          <TableRow
                            key={staff.id}
                            className="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-800"
                            onClick={() => handleStaffSelect(staff.id)}
                          >
                            <TableCell className="font-medium">
                              <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#005a87] text-white">
                                  {staff.first_name.charAt(0)}
                                  {staff.last_name.charAt(0)}
                                </div>
                                <div>
                                  <div>
                                    {staff.first_name} {staff.last_name}
                                  </div>
                                  <div className="text-sm text-gray-500">
                                    {staff.role === "teacher"
                                      ? "Öğretim Üyesi"
                                      : staff.role}
                                  </div>
                                </div>
                              </div>
                            </TableCell>
                            <TableCell className="text-gray-600 dark:text-gray-400">
                              {staff.email}
                            </TableCell>
                            <TableCell className="text-gray-600 dark:text-gray-400">
                              {staff.office_location || "-"}
                            </TableCell>
                            <TableCell>
                              <span
                                className={`rounded-full px-2 py-1 text-xs ${
                                  staff.status === "active"
                                    ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                                    : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                                }`}
                              >
                                {staff.status === "active" ? "Aktif" : "Pasif"}
                              </span>
                            </TableCell>
                            <TableCell className="text-right">
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  handleStaffSelect(staff.id)
                                }}
                              >
                                <User className="mr-1 h-4 w-4" />
                                Profil
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </>
              )}

              {/* Administrative Staff List */}
              {staffType === "administrative" && (
                <>
                  {adminStaffList.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">
                      <UserCog className="mx-auto mb-4 h-12 w-12 opacity-50" />
                      <p>Bu fakültede kayıtlı idari personel bulunamadı.</p>
                      <div className="mt-4 flex justify-center gap-3">
                        <Button
                          variant="outline"
                          onClick={handleBackToSelection}
                        >
                          Farklı Fakülte Seç
                        </Button>
                        <Button
                          onClick={handleOpenCreateDialog}
                          className="bg-[#005a87] text-white hover:bg-[#004a6d]"
                        >
                          <Plus className="mr-2 h-4 w-4" />
                          Yeni İdari Personel Ekle
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-gray-50 dark:bg-gray-800">
                          <TableHead className="font-semibold">
                            Ad Soyad
                          </TableHead>
                          <TableHead className="font-semibold">
                            Pozisyon
                          </TableHead>
                          <TableHead className="font-semibold">
                            E-posta
                          </TableHead>
                          <TableHead className="font-semibold">Durum</TableHead>
                          <TableHead className="text-right font-semibold">
                            İşlem
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {adminStaffList.map((staff) => (
                          <TableRow
                            key={staff.id}
                            className="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-800"
                            onClick={() => handleAdminStaffSelect(staff.id)}
                          >
                            <TableCell className="font-medium">
                              <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-emerald-600 text-white">
                                  {staff.first_name.charAt(0)}
                                  {staff.last_name.charAt(0)}
                                </div>
                                <div>
                                  <div>
                                    {staff.first_name} {staff.last_name}
                                  </div>
                                  <div className="text-sm text-gray-500">
                                    İdari Personel
                                  </div>
                                </div>
                              </div>
                            </TableCell>
                            <TableCell className="text-gray-600 dark:text-gray-400">
                              {staff.position}
                            </TableCell>
                            <TableCell className="text-gray-600 dark:text-gray-400">
                              {staff.email}
                            </TableCell>
                            <TableCell>
                              <span
                                className={`rounded-full px-2 py-1 text-xs ${
                                  staff.status === "active"
                                    ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                                    : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                                }`}
                              >
                                {staff.status === "active" ? "Aktif" : "Pasif"}
                              </span>
                            </TableCell>
                            <TableCell className="text-right">
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  handleAdminStaffSelect(staff.id)
                                }}
                              >
                                <User className="mr-1 h-4 w-4" />
                                Detay
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* ========== PROFILE VIEW ========== */}
      {viewMode === "profile" && (
        <>
          {/* Loading state */}
          {isLoading && (
            <div className="flex min-h-screen items-center justify-center">
              <div className="text-center">
                <div className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-b-2 border-[#005a87]"></div>
                <p className="text-gray-600 dark:text-gray-400">
                  Yükleniyor...
                </p>
              </div>
            </div>
          )}

          {/* Error state */}
          {!isLoading && !profile && !adminProfile && (
            <div className="flex min-h-screen items-center justify-center">
              <div className="text-center">
                <p className="text-red-600 dark:text-red-400">
                  Profil yüklenemedi
                </p>
                <Button onClick={handleBackToList} className="mt-4">
                  Listeye Dön
                </Button>
              </div>
            </div>
          )}

          {/* Administrative Staff Profile */}
          {!isLoading && adminProfile && (
            <>
              {/* Back button */}
              <div className="border-b bg-gray-100 dark:bg-gray-900">
                <div className="container mx-auto px-4 py-2">
                  <Button
                    variant="ghost"
                    onClick={handleBackToList}
                    className="text-emerald-600"
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Listeye Dön
                  </Button>
                </div>
              </div>

              {/* Admin Staff Header */}
              <div className="bg-emerald-600 text-white">
                <div className="container mx-auto px-4 py-6">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-6">
                      {/* Profil Resmi */}
                      <div className="group relative">
                        <div className="h-28 w-28 overflow-hidden rounded-lg border-4 border-white/30 bg-white/10 shadow-lg">
                          {adminProfile.profileImage ? (
                            <img
                              src={adminProfile.profileImage}
                              alt={`${adminProfile.firstName} ${adminProfile.lastName}`}
                              className="h-full w-full object-cover"
                            />
                          ) : (
                            <div className="flex h-full w-full items-center justify-center">
                              <UserCog className="h-16 w-16 text-white/60" />
                            </div>
                          )}
                        </div>
                        {isEditing && (
                          <button
                            type="button"
                            className="absolute inset-0 flex items-center justify-center rounded-lg bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
                            onClick={() => {
                              const url = prompt(
                                "Profil resmi URL'si girin:",
                                adminProfile.profileImage
                              )
                              if (url !== null) {
                                handleAdminInfoChange("profileImage", url)
                              }
                            }}
                          >
                            <Camera className="h-8 w-8 text-white" />
                          </button>
                        )}
                      </div>

                      {/* Info */}
                      <div className="flex-1">
                        <h1 className="text-2xl font-bold">
                          {adminProfile.firstName.toUpperCase()}{" "}
                          {adminProfile.lastName.toUpperCase()}
                        </h1>
                        <p className="mt-1 text-lg font-medium text-emerald-100">
                          {adminProfile.position}
                        </p>
                        <p className="text-sm text-emerald-200 italic">
                          {adminProfile.faculty}
                        </p>
                        <div className="mt-4 flex items-center gap-6 text-sm">
                          <div className="flex items-center gap-2">
                            <Mail className="h-4 w-4" />
                            <span>{adminProfile.email}</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Phone className="h-4 w-4" />
                            <span>{adminProfile.phone}</span>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Action Buttons */}
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="border-white/20 bg-white/10 text-white hover:bg-white/20"
                      >
                        <Printer className="mr-2 h-4 w-4" />
                        Yazdır
                      </Button>
                      {isAuthenticated &&
                        (!isEditing ? (
                          <Button
                            size="sm"
                            onClick={() => setIsEditing(true)}
                            className="bg-white text-emerald-600 hover:bg-emerald-50"
                          >
                            Düzenle
                          </Button>
                        ) : (
                          <Button
                            size="sm"
                            onClick={handleSave}
                            disabled={isSaving}
                            className="border border-emerald-700 bg-emerald-800 hover:bg-emerald-900"
                          >
                            <Save className="mr-2 h-4 w-4" />
                            {isSaving ? "Kaydediliyor..." : "Kaydet"}
                          </Button>
                        ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Admin Staff Basic Info Edit (when editing) */}
              {isEditing && (
                <div className="container mx-auto px-4 py-4">
                  <Card>
                    <CardContent className="pt-4">
                      <h3 className="mb-4 font-semibold text-emerald-700">
                        Temel Bilgiler
                      </h3>
                      <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                        <div className="space-y-2">
                          <Label>Ad</Label>
                          <Input
                            value={adminProfile.firstName}
                            onChange={(e) =>
                              handleAdminInfoChange("firstName", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Soyad</Label>
                          <Input
                            value={adminProfile.lastName}
                            onChange={(e) =>
                              handleAdminInfoChange("lastName", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Pozisyon</Label>
                          <Input
                            value={adminProfile.position}
                            onChange={(e) =>
                              handleAdminInfoChange("position", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Telefon</Label>
                          <Input
                            value={adminProfile.phone}
                            onChange={(e) =>
                              handleAdminInfoChange("phone", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>E-posta</Label>
                          <Input
                            value={adminProfile.email}
                            onChange={(e) =>
                              handleAdminInfoChange("email", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>Ofis Konumu</Label>
                          <Input
                            value={adminProfile.officeLocation}
                            onChange={(e) =>
                              handleAdminInfoChange(
                                "officeLocation",
                                e.target.value
                              )
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>Çalışma Saatleri</Label>
                          <Input
                            value={adminProfile.workingHours}
                            onChange={(e) =>
                              handleAdminInfoChange(
                                "workingHours",
                                e.target.value
                              )
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>Göreve Başlama</Label>
                          <Input
                            type="date"
                            value={adminProfile.startDate}
                            onChange={(e) =>
                              handleAdminInfoChange("startDate", e.target.value)
                            }
                          />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              )}

              {/* Admin Staff Details */}
              <div className="container mx-auto space-y-6 px-4 py-6">
                {/* Job Description */}
                <Card>
                  <CardHeader className="bg-emerald-50 dark:bg-emerald-900/20">
                    <CardTitle className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300">
                      <Briefcase className="h-5 w-5" />
                      Görev Tanımı
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-4">
                    {isEditing ? (
                      <textarea
                        className="min-h-[150px] w-full rounded-md border bg-white p-3 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500 dark:bg-gray-900"
                        value={adminProfile.jobDescription}
                        onChange={(e) =>
                          handleAdminInfoChange(
                            "jobDescription",
                            e.target.value
                          )
                        }
                        placeholder="Görev tanımını buraya giriniz..."
                      />
                    ) : (
                      <p className="leading-relaxed whitespace-pre-wrap text-gray-700 dark:text-gray-300">
                        {adminProfile.jobDescription}
                      </p>
                    )}
                  </CardContent>
                </Card>

                {/* Responsibilities */}
                <Card>
                  <CardHeader className="bg-emerald-50 dark:bg-emerald-900/20">
                    <div className="flex items-center justify-between">
                      <CardTitle className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300">
                        <ClipboardList className="h-5 w-5" />
                        Sorumluluklar
                      </CardTitle>
                      {isEditing && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={addAdminResponsibility}
                          className="border-emerald-200 text-emerald-600 hover:bg-emerald-50"
                        >
                          <Plus className="mr-2 h-4 w-4" />
                          Ekle
                        </Button>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent className="pt-4">
                    <ul className="space-y-2">
                      {adminProfile.responsibilities.map((resp, index) => (
                        <li
                          key={index}
                          className="flex items-start gap-2 text-gray-700 dark:text-gray-300"
                        >
                          <span
                            className={`${isEditing ? "mt-3" : "mt-2"} h-2 w-2 flex-shrink-0 rounded-full bg-emerald-500`}
                          ></span>
                          {isEditing ? (
                            <div className="flex flex-1 gap-2">
                              <Input
                                value={resp}
                                onChange={(e) =>
                                  handleAdminResponsibilityChange(
                                    index,
                                    e.target.value
                                  )
                                }
                                className="flex-1"
                              />
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => removeAdminResponsibility(index)}
                                className="h-10 w-10 p-2 text-red-500 hover:bg-red-50 hover:text-red-700"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          ) : (
                            <span>{resp}</span>
                          )}
                        </li>
                      ))}
                    </ul>
                    {adminProfile.responsibilities.length === 0 && (
                      <p className="text-gray-500 italic">
                        Henüz sorumluluk eklenmemiş.
                      </p>
                    )}
                  </CardContent>
                </Card>

                {/* Work Info Grid - Only shown when NOT editing since they are editable in the top form */}
                {!isEditing && (
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                    <Card>
                      <CardContent className="pt-4">
                        <div className="flex items-center gap-3">
                          <div className="rounded-full bg-emerald-100 p-3 dark:bg-emerald-900">
                            <MapPin className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-500">Ofis Konumu</p>
                            <p className="font-medium text-gray-900 dark:text-gray-100">
                              {adminProfile.officeLocation}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardContent className="pt-4">
                        <div className="flex items-center gap-3">
                          <div className="rounded-full bg-emerald-100 p-3 dark:bg-emerald-900">
                            <Clock className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-500">
                              Çalışma Saatleri
                            </p>
                            <p className="font-medium text-gray-900 dark:text-gray-100">
                              {adminProfile.workingHours}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardContent className="pt-4">
                        <div className="flex items-center gap-3">
                          <div className="rounded-full bg-emerald-100 p-3 dark:bg-emerald-900">
                            <Calendar className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-500">
                              Göreve Başlama
                            </p>
                            <p className="font-medium text-gray-900 dark:text-gray-100">
                              {new Date(
                                adminProfile.startDate
                              ).toLocaleDateString("tr-TR", {
                                year: "numeric",
                                month: "long",
                                day: "numeric",
                              })}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                )}
              </div>
            </>
          )}

          {/* Academic Staff Profile - Main content - only render when profile is loaded */}
          {!isLoading && profile && (
            <>
              {/* Back button */}
              <div className="border-b bg-gray-100 dark:bg-gray-900">
                <div className="container mx-auto px-4 py-2">
                  <Button
                    variant="ghost"
                    onClick={handleBackToList}
                    className="text-[#005a87]"
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Listeye Dön
                  </Button>
                </div>
              </div>
              {/* Header - MDC Style */}
              <div className="bg-[#005a87] text-white">
                <div className="container mx-auto px-4 py-6">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-6">
                      {/* Profil Resmi */}
                      <div className="group relative">
                        <div className="h-28 w-28 overflow-hidden rounded-lg border-4 border-white/30 bg-white/10 shadow-lg">
                          {profile.profileImage ? (
                            <img
                              src={profile.profileImage}
                              alt={`${profile.firstName} ${profile.lastName}`}
                              className="h-full w-full object-cover"
                            />
                          ) : (
                            <div className="flex h-full w-full items-center justify-center">
                              <User className="h-16 w-16 text-white/60" />
                            </div>
                          )}
                        </div>
                        {isEditing && (
                          <button
                            type="button"
                            className="absolute inset-0 flex items-center justify-center rounded-lg bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
                            onClick={() => {
                              const url = prompt(
                                "Profil resmi URL'si girin:",
                                profile.profileImage
                              )
                              if (url !== null) {
                                handleBasicInfoChange("profileImage", url)
                              }
                            }}
                          >
                            <Camera className="h-8 w-8 text-white" />
                          </button>
                        )}
                      </div>
                      {/* Bilgiler */}
                      <div>
                        <h1 className="text-2xl font-bold">
                          {profile.title} {profile.firstName.toUpperCase()}{" "}
                          {profile.lastName.toUpperCase()}
                        </h1>
                        <p className="mt-1 text-sm text-blue-100 italic">
                          {profile.faculty} {profile.department}
                        </p>
                        <div className="mt-4 flex items-center gap-6 text-sm">
                          <div className="flex items-center gap-2">
                            <Mail className="h-4 w-4" />
                            <span>{profile.email}</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Phone className="h-4 w-4" />
                            <span>{profile.phone}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="border-white/20 bg-white/10 text-white hover:bg-white/20"
                      >
                        <Printer className="mr-2 h-4 w-4" />
                        Yazdır
                      </Button>
                      {isAuthenticated &&
                        (!isEditing ? (
                          <Button
                            size="sm"
                            onClick={() => setIsEditing(true)}
                            className="bg-white text-[#005a87] hover:bg-gray-100"
                          >
                            Düzenle
                          </Button>
                        ) : (
                          <Button
                            size="sm"
                            onClick={handleSave}
                            disabled={isSaving}
                            className="bg-green-600 hover:bg-green-700"
                          >
                            <Save className="mr-2 h-4 w-4" />
                            {isSaving ? "Kaydediliyor..." : "Kaydet"}
                          </Button>
                        ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Basic Info Edit (when editing) */}
              {isEditing && (
                <div className="container mx-auto px-4 py-4">
                  <Card>
                    <CardContent className="pt-4">
                      <h3 className="mb-4 font-semibold">Temel Bilgiler</h3>
                      <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                        <div className="space-y-2">
                          <Label>Unvan</Label>
                          <Select
                            value={profile.title}
                            onValueChange={(v) =>
                              handleBasicInfoChange("title", v)
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="Prof. Dr.">
                                Prof. Dr.
                              </SelectItem>
                              <SelectItem value="Doç. Dr.">Doç. Dr.</SelectItem>
                              <SelectItem value="Dr. Öğr. Üyesi">
                                Dr. Öğr. Üyesi
                              </SelectItem>
                              <SelectItem value="Öğr. Gör.">
                                Öğr. Gör.
                              </SelectItem>
                              <SelectItem value="Arş. Gör.">
                                Arş. Gör.
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-2">
                          <Label>Ad</Label>
                          <Input
                            value={profile.firstName}
                            onChange={(e) =>
                              handleBasicInfoChange("firstName", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Soyad</Label>
                          <Input
                            value={profile.lastName}
                            onChange={(e) =>
                              handleBasicInfoChange("lastName", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>E-posta</Label>
                          <Input
                            value={profile.email}
                            onChange={(e) =>
                              handleBasicInfoChange("email", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Telefon</Label>
                          <Input
                            value={profile.phone}
                            onChange={(e) =>
                              handleBasicInfoChange("phone", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>Fakülte</Label>
                          <Input
                            value={profile.faculty}
                            onChange={(e) =>
                              handleBasicInfoChange("faculty", e.target.value)
                            }
                          />
                        </div>
                        <div className="space-y-2 md:col-span-2">
                          <Label>Bölüm</Label>
                          <Input
                            value={profile.department}
                            onChange={(e) =>
                              handleBasicInfoChange(
                                "department",
                                e.target.value
                              )
                            }
                          />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              )}

              {/* Tabs Content */}
              <div className="container mx-auto px-4 py-6">
                <Tabs value={activeTab} onValueChange={setActiveTab}>
                  <TabsList className="mb-6 h-auto flex-wrap border bg-white p-1 dark:bg-gray-900">
                    <TabsTrigger value="education" className="gap-2">
                      <GraduationCap className="h-4 w-4" />
                      EĞİTİM
                    </TabsTrigger>
                    <TabsTrigger value="articles" className="gap-2">
                      <FileText className="h-4 w-4" />
                      MAKALELER
                    </TabsTrigger>
                    <TabsTrigger value="bulletins" className="gap-2">
                      <BookOpen className="h-4 w-4" />
                      BİLDİRİLER
                    </TabsTrigger>
                    <TabsTrigger value="projects" className="gap-2">
                      <Briefcase className="h-4 w-4" />
                      PROJELER
                    </TabsTrigger>
                    <TabsTrigger value="awards" className="gap-2">
                      <Award className="h-4 w-4" />
                      ÖDÜLLER
                    </TabsTrigger>
                    <TabsTrigger value="scholarships">BURSLAR</TabsTrigger>
                    <TabsTrigger value="admin">
                      AKADEMİK & İDARİ GÖREVLER
                    </TabsTrigger>
                  </TabsList>

                  {/* Education Tab */}
                  <TabsContent value="education">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">
                            Eğitim / Akademik Bilgileri
                          </h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addEducation}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Ekle
                            </Button>
                          )}
                        </div>
                        <div className="overflow-x-auto">
                          <table className="w-full">
                            <thead>
                              <tr className="border-b">
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Derece
                                </th>
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Kurum
                                </th>
                                <th className="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">
                                  Yıl
                                </th>
                                {isEditing && <th className="w-10"></th>}
                              </tr>
                            </thead>
                            <tbody>
                              {profile.education.map((edu) => (
                                <tr
                                  key={edu.id}
                                  className="border-b last:border-0 hover:bg-gray-50 dark:hover:bg-gray-900"
                                >
                                  <td className="px-4 py-3">
                                    {isEditing ? (
                                      <Input
                                        value={edu.degree}
                                        onChange={(e) =>
                                          updateEducation(
                                            edu.id,
                                            "degree",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      edu.degree
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-blue-600 dark:text-blue-400">
                                    {isEditing ? (
                                      <Input
                                        value={edu.institution}
                                        onChange={(e) =>
                                          updateEducation(
                                            edu.id,
                                            "institution",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      edu.institution
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-right">
                                    {isEditing ? (
                                      <Input
                                        type="number"
                                        value={edu.year}
                                        onChange={(e) =>
                                          updateEducation(
                                            edu.id,
                                            "year",
                                            parseInt(e.target.value)
                                          )
                                        }
                                        className="h-8 w-24 text-right"
                                      />
                                    ) : (
                                      edu.year
                                    )}
                                  </td>
                                  {isEditing && (
                                    <td className="px-2 py-3">
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => removeEducation(edu.id)}
                                        className="text-destructive hover:text-destructive"
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </td>
                                  )}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Articles Tab */}
                  <TabsContent value="articles">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Makaleler</h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addArticle}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Makale Ekle
                            </Button>
                          )}
                        </div>
                        {isEditing ? (
                          <div className="space-y-4">
                            {profile.articles.map((article, index) => (
                              <div
                                key={article.id}
                                className="space-y-3 rounded-lg border p-4"
                              >
                                <div className="flex justify-between">
                                  <span className="text-sm text-muted-foreground">
                                    Makale #{index + 1}
                                  </span>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => removeArticle(article.id)}
                                    className="text-destructive hover:text-destructive"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </div>
                                <Input
                                  placeholder="Makale Başlığı"
                                  value={article.title}
                                  onChange={(e) =>
                                    updateArticle(
                                      article.id,
                                      "title",
                                      e.target.value
                                    )
                                  }
                                />
                                <div className="grid grid-cols-2 gap-3">
                                  <Input
                                    placeholder="Dergi Adı"
                                    value={article.journal}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "journal",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <Input
                                    type="number"
                                    placeholder="Yıl"
                                    value={article.year}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "year",
                                        parseInt(e.target.value)
                                      )
                                    }
                                  />
                                </div>
                                <Input
                                  placeholder="Yazarlar"
                                  value={article.authors}
                                  onChange={(e) =>
                                    updateArticle(
                                      article.id,
                                      "authors",
                                      e.target.value
                                    )
                                  }
                                />
                                <div className="grid grid-cols-2 gap-3">
                                  <Input
                                    placeholder="Dergi Tipi (SCI, SCI-Expanded, vb.)"
                                    value={article.journalType || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "journalType",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <Input
                                    placeholder="Yurtiçi/Yurtdışı"
                                    value={article.domesticInternational || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "domesticInternational",
                                        e.target.value
                                      )
                                    }
                                  />
                                </div>
                                <div className="grid grid-cols-3 gap-3">
                                  <Input
                                    placeholder="Yayın Ayı/Yılı"
                                    value={article.publishingMonth || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "publishingMonth",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <Input
                                    placeholder="Cilt/Sayı/Sayfa"
                                    value={article.issuePageYear || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "issuePageYear",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <Input
                                    placeholder="Dil"
                                    value={article.language || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "language",
                                        e.target.value
                                      )
                                    }
                                  />
                                </div>
                                <div className="grid grid-cols-2 gap-3">
                                  <Input
                                    placeholder="Makale Tipi"
                                    value={article.articleType || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "articleType",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <Input
                                    placeholder="DOI (opsiyonel)"
                                    value={article.doi || ""}
                                    onChange={(e) =>
                                      updateArticle(
                                        article.id,
                                        "doi",
                                        e.target.value
                                      )
                                    }
                                  />
                                </div>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <div className="space-y-2">
                            {profile.articles.map((article, index) => {
                              const isExpanded = expandedArticles.includes(
                                article.id
                              )
                              return (
                                <div
                                  key={article.id}
                                  className="overflow-hidden rounded-lg border"
                                >
                                  {/* Article Header - Always Visible */}
                                  <div
                                    onClick={() => {
                                      setExpandedArticles((prev) =>
                                        prev.includes(article.id)
                                          ? prev.filter(
                                              (id) => id !== article.id
                                            )
                                          : [...prev, article.id]
                                      )
                                    }}
                                    className="flex cursor-pointer items-center justify-between p-4 transition-colors hover:bg-blue-50 dark:hover:bg-blue-900/20"
                                  >
                                    <div className="flex flex-1 items-start gap-4">
                                      <span className="min-w-[30px] font-medium text-gray-500">
                                        {index + 1}
                                      </span>
                                      <div className="flex-1">
                                        <h4 className="font-medium text-blue-600 dark:text-blue-400">
                                          {article.title}
                                        </h4>
                                        <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                                          {article.journal} • {article.year}
                                        </p>
                                      </div>
                                    </div>
                                    <ChevronDown
                                      className={`h-5 w-5 text-gray-500 transition-transform duration-200 ${
                                        isExpanded ? "rotate-180 transform" : ""
                                      }`}
                                    />
                                  </div>

                                  {/* Article Details - Expandable */}
                                  {isExpanded && (
                                    <div className="border-t bg-gray-50 p-4 dark:bg-gray-900">
                                      <div className="grid grid-cols-1 gap-3 text-sm md:grid-cols-2">
                                        <div>
                                          <span className="font-semibold text-gray-900 dark:text-gray-100">
                                            JOURNAL NAME:
                                          </span>
                                          <p className="mt-1 text-gray-700 dark:text-gray-300">
                                            {article.journal}
                                          </p>
                                        </div>
                                        {article.journalType && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              JOURNAL TYPE:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.journalType}
                                            </p>
                                          </div>
                                        )}
                                        {article.domesticInternational && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              DOMESTIC/INTERNATIONAL:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.domesticInternational}
                                            </p>
                                          </div>
                                        )}
                                        {article.publishingMonth && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              PUBLISHING MONTH/YEAR:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.publishingMonth}
                                            </p>
                                          </div>
                                        )}
                                        {article.issuePageYear && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              ISSUE/PAGE/YEAR:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.issuePageYear}
                                            </p>
                                          </div>
                                        )}
                                        {article.language && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              LANGUAGE:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.language}
                                            </p>
                                          </div>
                                        )}
                                        {article.articleType && (
                                          <div>
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              ARTICLE TYPE:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.articleType}
                                            </p>
                                          </div>
                                        )}
                                        <div className="md:col-span-2">
                                          <span className="font-semibold text-gray-900 dark:text-gray-100">
                                            AUTHORS:
                                          </span>
                                          <p className="mt-1 text-gray-700 dark:text-gray-300">
                                            {article.authors}
                                          </p>
                                        </div>
                                        {article.doi && (
                                          <div className="md:col-span-2">
                                            <span className="font-semibold text-gray-900 dark:text-gray-100">
                                              DOI:
                                            </span>
                                            <p className="mt-1 text-gray-700 dark:text-gray-300">
                                              {article.doi}
                                            </p>
                                          </div>
                                        )}
                                      </div>
                                    </div>
                                  )}
                                </div>
                              )
                            })}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Bulletins Tab */}
                  <TabsContent value="bulletins">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Bildiriler</h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addBulletin}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Bildiri Ekle
                            </Button>
                          )}
                        </div>
                        <div className="space-y-4">
                          {profile.bulletins.map((bulletin, index) => (
                            <div
                              key={bulletin.id}
                              className="rounded-lg border p-4"
                            >
                              {isEditing ? (
                                <div className="space-y-3">
                                  <div className="flex justify-between">
                                    <span className="text-sm text-muted-foreground">
                                      Bildiri #{index + 1}
                                    </span>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() =>
                                        removeBulletin(bulletin.id)
                                      }
                                      className="text-destructive hover:text-destructive"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </Button>
                                  </div>
                                  <Input
                                    placeholder="Bildiri Başlığı"
                                    value={bulletin.title}
                                    onChange={(e) =>
                                      updateBulletin(
                                        bulletin.id,
                                        "title",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <div className="grid grid-cols-2 gap-3">
                                    <Input
                                      placeholder="Konferans Adı"
                                      value={bulletin.conference}
                                      onChange={(e) =>
                                        updateBulletin(
                                          bulletin.id,
                                          "conference",
                                          e.target.value
                                        )
                                      }
                                    />
                                    <Input
                                      type="number"
                                      placeholder="Yıl"
                                      value={bulletin.year}
                                      onChange={(e) =>
                                        updateBulletin(
                                          bulletin.id,
                                          "year",
                                          parseInt(e.target.value)
                                        )
                                      }
                                    />
                                  </div>
                                  <Input
                                    placeholder="Yer"
                                    value={bulletin.location}
                                    onChange={(e) =>
                                      updateBulletin(
                                        bulletin.id,
                                        "location",
                                        e.target.value
                                      )
                                    }
                                  />
                                </div>
                              ) : (
                                <div>
                                  <h4 className="font-medium text-blue-600 dark:text-blue-400">
                                    {bulletin.title}
                                  </h4>
                                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                                    {bulletin.conference}, {bulletin.year}
                                  </p>
                                  <p className="mt-1 text-sm text-gray-500">
                                    {bulletin.location}
                                  </p>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Projects Tab */}
                  <TabsContent value="projects">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Projeler</h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addProject}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Proje Ekle
                            </Button>
                          )}
                        </div>
                        <div className="space-y-4">
                          {profile.projects.map((project, index) => (
                            <div
                              key={project.id}
                              className="rounded-lg border p-4"
                            >
                              {isEditing ? (
                                <div className="space-y-3">
                                  <div className="flex justify-between">
                                    <span className="text-sm text-muted-foreground">
                                      Proje #{index + 1}
                                    </span>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() => removeProject(project.id)}
                                      className="text-destructive hover:text-destructive"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </Button>
                                  </div>
                                  <Input
                                    placeholder="Proje Başlığı"
                                    value={project.title}
                                    onChange={(e) =>
                                      updateProject(
                                        project.id,
                                        "title",
                                        e.target.value
                                      )
                                    }
                                  />
                                  <div className="grid grid-cols-2 gap-3">
                                    <Input
                                      placeholder="Rol"
                                      value={project.role}
                                      onChange={(e) =>
                                        updateProject(
                                          project.id,
                                          "role",
                                          e.target.value
                                        )
                                      }
                                    />
                                    <Input
                                      placeholder="Destekleyen Kuruluş"
                                      value={project.funder}
                                      onChange={(e) =>
                                        updateProject(
                                          project.id,
                                          "funder",
                                          e.target.value
                                        )
                                      }
                                    />
                                  </div>
                                  <div className="grid grid-cols-3 gap-3">
                                    <Input
                                      type="number"
                                      placeholder="Başlangıç Yılı"
                                      value={project.startYear}
                                      onChange={(e) =>
                                        updateProject(
                                          project.id,
                                          "startYear",
                                          parseInt(e.target.value)
                                        )
                                      }
                                    />
                                    <Input
                                      type="number"
                                      placeholder="Bitiş Yılı"
                                      value={project.endYear || ""}
                                      onChange={(e) =>
                                        updateProject(
                                          project.id,
                                          "endYear",
                                          e.target.value
                                            ? parseInt(e.target.value)
                                            : ""
                                        )
                                      }
                                    />
                                    <Select
                                      value={project.status}
                                      onValueChange={(v) =>
                                        updateProject(project.id, "status", v)
                                      }
                                    >
                                      <SelectTrigger>
                                        <SelectValue />
                                      </SelectTrigger>
                                      <SelectContent>
                                        <SelectItem value="ongoing">
                                          Devam Ediyor
                                        </SelectItem>
                                        <SelectItem value="completed">
                                          Tamamlandı
                                        </SelectItem>
                                      </SelectContent>
                                    </Select>
                                  </div>
                                </div>
                              ) : (
                                <div>
                                  <div className="flex items-center gap-2">
                                    <h4 className="font-medium text-blue-600 dark:text-blue-400">
                                      {project.title}
                                    </h4>
                                    <span
                                      className={`rounded px-2 py-0.5 text-xs ${project.status === "ongoing" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-700"}`}
                                    >
                                      {project.status === "ongoing"
                                        ? "Devam Ediyor"
                                        : "Tamamlandı"}
                                    </span>
                                  </div>
                                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                                    {project.role} - {project.funder}
                                  </p>
                                  <p className="mt-1 text-sm text-gray-500">
                                    {project.startYear} -{" "}
                                    {project.endYear || "Devam"}
                                  </p>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Awards Tab */}
                  <TabsContent value="awards">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Ödüller</h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addAward}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Ödül Ekle
                            </Button>
                          )}
                        </div>
                        <div className="overflow-x-auto">
                          <table className="w-full">
                            <thead>
                              <tr className="border-b">
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Ödül Adı
                                </th>
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Veren Kurum
                                </th>
                                <th className="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">
                                  Yıl
                                </th>
                                {isEditing && <th className="w-10"></th>}
                              </tr>
                            </thead>
                            <tbody>
                              {profile.awards.map((award) => (
                                <tr
                                  key={award.id}
                                  className="border-b last:border-0 hover:bg-gray-50 dark:hover:bg-gray-900"
                                >
                                  <td className="px-4 py-3">
                                    {isEditing ? (
                                      <Input
                                        value={award.title}
                                        onChange={(e) =>
                                          updateAward(
                                            award.id,
                                            "title",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      award.title
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-blue-600 dark:text-blue-400">
                                    {isEditing ? (
                                      <Input
                                        value={award.institution}
                                        onChange={(e) =>
                                          updateAward(
                                            award.id,
                                            "institution",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      award.institution
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-right">
                                    {isEditing ? (
                                      <Input
                                        type="number"
                                        value={award.year}
                                        onChange={(e) =>
                                          updateAward(
                                            award.id,
                                            "year",
                                            parseInt(e.target.value)
                                          )
                                        }
                                        className="h-8 w-24 text-right"
                                      />
                                    ) : (
                                      award.year
                                    )}
                                  </td>
                                  {isEditing && (
                                    <td className="px-2 py-3">
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => removeAward(award.id)}
                                        className="text-destructive hover:text-destructive"
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </td>
                                  )}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Scholarships Tab */}
                  <TabsContent value="scholarships">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Burslar</h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addScholarship}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Burs Ekle
                            </Button>
                          )}
                        </div>
                        <div className="overflow-x-auto">
                          <table className="w-full">
                            <thead>
                              <tr className="border-b">
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Burs Adı
                                </th>
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Veren Kurum
                                </th>
                                <th className="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">
                                  Yıl
                                </th>
                                {isEditing && <th className="w-10"></th>}
                              </tr>
                            </thead>
                            <tbody>
                              {profile.scholarships.map((scholarship) => (
                                <tr
                                  key={scholarship.id}
                                  className="border-b last:border-0 hover:bg-gray-50 dark:hover:bg-gray-900"
                                >
                                  <td className="px-4 py-3">
                                    {isEditing ? (
                                      <Input
                                        value={scholarship.title}
                                        onChange={(e) =>
                                          updateScholarship(
                                            scholarship.id,
                                            "title",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      scholarship.title
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-blue-600 dark:text-blue-400">
                                    {isEditing ? (
                                      <Input
                                        value={scholarship.institution}
                                        onChange={(e) =>
                                          updateScholarship(
                                            scholarship.id,
                                            "institution",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      scholarship.institution
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-right">
                                    {isEditing ? (
                                      <Input
                                        type="number"
                                        value={scholarship.year}
                                        onChange={(e) =>
                                          updateScholarship(
                                            scholarship.id,
                                            "year",
                                            parseInt(e.target.value)
                                          )
                                        }
                                        className="h-8 w-24 text-right"
                                      />
                                    ) : (
                                      scholarship.year
                                    )}
                                  </td>
                                  {isEditing && (
                                    <td className="px-2 py-3">
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() =>
                                          removeScholarship(scholarship.id)
                                        }
                                        className="text-destructive hover:text-destructive"
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </td>
                                  )}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Admin Assignments Tab */}
                  <TabsContent value="admin">
                    <Card>
                      <CardContent className="pt-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold">
                            Akademik & İdari Görevler
                          </h3>
                          {isEditing && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={addAdminAssignment}
                            >
                              <Plus className="mr-2 h-4 w-4" />
                              Görev Ekle
                            </Button>
                          )}
                        </div>
                        <div className="overflow-x-auto">
                          <table className="w-full">
                            <thead>
                              <tr className="border-b">
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Görev
                                </th>
                                <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">
                                  Kurum
                                </th>
                                <th className="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">
                                  Dönem
                                </th>
                                {isEditing && <th className="w-10"></th>}
                              </tr>
                            </thead>
                            <tbody>
                              {profile.adminAssignments.map((assignment) => (
                                <tr
                                  key={assignment.id}
                                  className="border-b last:border-0 hover:bg-gray-50 dark:hover:bg-gray-900"
                                >
                                  <td className="px-4 py-3">
                                    {isEditing ? (
                                      <Input
                                        value={assignment.title}
                                        onChange={(e) =>
                                          updateAdminAssignment(
                                            assignment.id,
                                            "title",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      assignment.title
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-blue-600 dark:text-blue-400">
                                    {isEditing ? (
                                      <Input
                                        value={assignment.institution}
                                        onChange={(e) =>
                                          updateAdminAssignment(
                                            assignment.id,
                                            "institution",
                                            e.target.value
                                          )
                                        }
                                        className="h-8"
                                      />
                                    ) : (
                                      assignment.institution
                                    )}
                                  </td>
                                  <td className="px-4 py-3 text-right">
                                    {isEditing ? (
                                      <div className="flex justify-end gap-2">
                                        <Input
                                          type="number"
                                          placeholder="Başlangıç"
                                          value={assignment.startYear}
                                          onChange={(e) =>
                                            updateAdminAssignment(
                                              assignment.id,
                                              "startYear",
                                              parseInt(e.target.value)
                                            )
                                          }
                                          className="h-8 w-20 text-right"
                                        />
                                        <span className="self-center">-</span>
                                        <Input
                                          type="number"
                                          placeholder="Bitiş"
                                          value={assignment.endYear || ""}
                                          onChange={(e) =>
                                            updateAdminAssignment(
                                              assignment.id,
                                              "endYear",
                                              e.target.value
                                                ? parseInt(e.target.value)
                                                : undefined
                                            )
                                          }
                                          className="h-8 w-20 text-right"
                                        />
                                      </div>
                                    ) : (
                                      `${assignment.startYear} - ${assignment.endYear || "Devam"}`
                                    )}
                                  </td>
                                  {isEditing && (
                                    <td className="px-2 py-3">
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() =>
                                          removeAdminAssignment(assignment.id)
                                        }
                                        className="text-destructive hover:text-destructive"
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </td>
                                  )}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>
                </Tabs>
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}
