"use client";

import { useState, useEffect, useMemo } from "react";
import { staffApi } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Plus, Pencil, Trash2, ArrowUpDown, ArrowUp, ArrowDown, Check, ChevronsUpDown } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { mockFaculties } from "@/mock_data/catalog";

interface Staff {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  faculty: string;
  department: string;
  phone: string;
  office_location: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface StaffListResponse {
  data: Staff[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

type SortField = 'first_name' | 'last_name' | 'email' | 'role' | 'department' | 'office_location';
type SortDirection = 'asc' | 'desc';

// Role options based on backend validation
const ROLE_OPTIONS = [
  { value: 'teacher', label: 'Teacher' },
  { value: 'admin', label: 'Admin' },
  { value: 'department_head', label: 'Department Head' },
  { value: 'dean', label: 'Dean' },
  { value: 'rector', label: 'Rector' },
  { value: 'advisor', label: 'Advisor' },
];

export default function StaffPage() {
  const [staffList, setStaffList] = useState<Staff[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingStaff, setEditingStaff] = useState<Staff | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [sortField, setSortField] = useState<SortField>('first_name');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  // UI state for faculty/department selection (faculty is only for filtering, not sent to backend)
  const [selectedFaculty, setSelectedFaculty] = useState("");
  const [facultyOpen, setFacultyOpen] = useState(false);
  const [departmentOpen, setDepartmentOpen] = useState(false);

  // UI state for edit form faculty/department selection
  const [editSelectedFaculty, setEditSelectedFaculty] = useState("");
  const [editFacultyOpen, setEditFacultyOpen] = useState(false);
  const [editDepartmentOpen, setEditDepartmentOpen] = useState(false);

  // Form states - Backend DTO'ya uygun alanlar
  const [formData, setFormData] = useState({
    email: "",
    first_name: "",
    last_name: "",
    role: "teacher",
    department: "",
    phone: "",
    office_location: "",
  });

  // Filtered departments based on selected faculty (for create form)
  const filteredDepartments = useMemo(() => {
    if (!selectedFaculty) return [];
    const faculty = mockFaculties.find(f => f.name === selectedFaculty);
    return faculty?.departments || [];
  }, [selectedFaculty]);

  // Filtered departments based on selected faculty (for edit form)
  const editFilteredDepartments = useMemo(() => {
    if (!editSelectedFaculty) return [];
    const faculty = mockFaculties.find(f => f.name === editSelectedFaculty);
    return faculty?.departments || [];
  }, [editSelectedFaculty]);

  // Update form for edit - tüm alanlar düzenlenebilir
  const [updateFormData, setUpdateFormData] = useState({
    email: "",
    first_name: "",
    last_name: "",
    department: "",
    phone: "",
    office_location: "",
    status: "active",
  });

  // Fetch staff list
  const fetchStaff = async (currentPage: number = 1) => {
    setLoading(true);
    setError("");
    try {
      const response: StaffListResponse = await staffApi
        .get(`?page=${currentPage}&limit=10`)
        .json();

      setStaffList(response.data);
      setPage(response.pagination.page);
      setTotalPages(response.pagination.total_pages);
    } catch (err: any) {
      setError(err.message || "Failed to fetch staff");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStaff();
  }, []);

  // Sorting fonksiyonu
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      // Aynı sütuna tıklandıysa direction değiştir
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      // Farklı sütuna tıklandıysa yeni field ve asc
      setSortField(field);
      setSortDirection('asc');
    }
  };

  // Sıralanmış staff listesi
  const sortedStaffList = [...staffList].sort((a, b) => {
    let aValue = a[sortField] || '';
    let bValue = b[sortField] || '';

    // String comparison
    if (typeof aValue === 'string' && typeof bValue === 'string') {
      aValue = aValue.toLowerCase();
      bValue = bValue.toLowerCase();
    }

    if (sortDirection === 'asc') {
      return aValue > bValue ? 1 : -1;
    } else {
      return aValue < bValue ? 1 : -1;
    }
  });

  // Create staff
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      // Backend DTO'ya uygun payload - sadece kabul edilen alanlar
      const payload = {
        email: formData.email,
        first_name: formData.first_name,
        last_name: formData.last_name,
        role: formData.role,
        department: formData.department,
        phone: formData.phone,
        office_location: formData.office_location,
      };
      
      console.log("[Staff] Creating staff with payload:", payload);
      
      const response = await staffApi.post("", { json: payload }).json();
      console.log("[Staff] Create response:", response);
      
      setCreateDialogOpen(false);
      setFormData({
        email: "",
        first_name: "",
        last_name: "",
        role: "teacher",
        department: "",
        phone: "",
        office_location: "",
      });
      setSelectedFaculty("");
      fetchStaff(page);
    } catch (err: any) {
      console.error("[Staff] Create error:", err);
      // Try to get error message from response
      let errorMessage = "Failed to create staff";
      if (err.response) {
        try {
          const errorBody = await err.response.json();
          errorMessage = errorBody.message || errorBody.error || errorMessage;
        } catch {
          errorMessage = err.message || errorMessage;
        }
      } else {
        errorMessage = err.message || errorMessage;
      }
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // Update staff
  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingStaff) return;

    setLoading(true);
    setError("");

    try {
      // Backend sadece bu alanları kabul ediyor
      const payload = {
        department: updateFormData.department,
        phone: updateFormData.phone,
        office_location: updateFormData.office_location,
      };

      console.log("[Staff] Updating staff with payload:", payload);
      
      const response = await staffApi.put(editingStaff.id, { json: payload }).json();
      console.log("[Staff] Update response:", response);
      
      setEditDialogOpen(false);
      setEditingStaff(null);
      setUpdateFormData({
        email: "",
        first_name: "",
        last_name: "",
        department: "",
        phone: "",
        office_location: "",
        status: "active",
      });
      setEditSelectedFaculty("");
      fetchStaff(page);
    } catch (err: any) {
      console.error("[Staff] Update error:", err);
      let errorMessage = "Failed to update staff";
      if (err.response) {
        try {
          const errorBody = await err.response.json();
          errorMessage = errorBody.message || errorBody.error || errorMessage;
        } catch {
          errorMessage = err.message || errorMessage;
        }
      } else {
        errorMessage = err.message || errorMessage;
      }
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // Delete staff
  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this staff member?")) return;

    setLoading(true);
    setError("");

    try {
      console.log("[Staff] Deleting staff:", id);
      await staffApi.delete(id).json();
      console.log("[Staff] Delete successful");
      fetchStaff(page);
    } catch (err: any) {
      console.error("[Staff] Delete error:", err);
      setError(err.message || "Failed to delete staff");
    } finally {
      setLoading(false);
    }
  };

  // Open edit dialog
  const openEditDialog = (staff: Staff) => {
    setEditingStaff(staff);
    setUpdateFormData({
      email: staff.email,
      first_name: staff.first_name,
      last_name: staff.last_name,
      department: staff.department || "",
      phone: staff.phone || "",
      office_location: staff.office_location || "",
      status: staff.status,
    });

    // Find the faculty that contains this department
    if (staff.department) {
      const facultyWithDept = mockFaculties.find(f =>
        f.departments.some(d => d.name === staff.department)
      );
      setEditSelectedFaculty(facultyWithDept?.name || "");
    } else {
      setEditSelectedFaculty("");
    }

    setEditDialogOpen(true);
  };

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold">Staff Management</h1>
            <p className="text-muted-foreground mt-1">
              Manage staff members and their information
            </p>
          </div>

          {/* Create Dialog */}
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Yeni Personel Ekle
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[500px]">
              <DialogHeader>
                <DialogTitle>Yeni Personel Ekle</DialogTitle>
              </DialogHeader>
              <form onSubmit={handleCreate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">E-posta *</Label>
                  <Input
                    id="email"
                    type="email"
                    required
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="first_name">Ad *</Label>
                    <Input
                      id="first_name"
                      required
                      value={formData.first_name}
                      onChange={(e) =>
                        setFormData({ ...formData, first_name: e.target.value })
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="last_name">Soyad *</Label>
                    <Input
                      id="last_name"
                      required
                      value={formData.last_name}
                      onChange={(e) =>
                        setFormData({ ...formData, last_name: e.target.value })
                      }
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="role">Rol *</Label>
                  <Select
                    value={formData.role}
                    onValueChange={(value) =>
                      setFormData({ ...formData, role: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Rol seçin" />
                    </SelectTrigger>
                    <SelectContent>
                      {ROLE_OPTIONS.map((role) => (
                        <SelectItem key={role.value} value={role.value}>
                          {role.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="faculty">Fakülte (filtreleme için)</Label>
                  <Popover open={facultyOpen} onOpenChange={setFacultyOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={facultyOpen}
                        className="w-full justify-between"
                      >
                        {selectedFaculty || "Fakülte seçin..."}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
                      <Command>
                        <CommandInput placeholder="Fakülte ara..." />
                        <CommandList>
                          <CommandEmpty>Fakülte bulunamadı.</CommandEmpty>
                          <CommandGroup>
                            {mockFaculties.map((faculty) => (
                              <CommandItem
                                key={faculty.id}
                                value={faculty.name}
                                onSelect={() => {
                                  setSelectedFaculty(selectedFaculty === faculty.name ? "" : faculty.name);
                                  setFormData({ ...formData, department: "" });
                                  setFacultyOpen(false);
                                }}
                              >
                                <Check
                                  className={`mr-2 h-4 w-4 ${selectedFaculty === faculty.name ? "opacity-100" : "opacity-0"}`}
                                />
                                {faculty.name}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="department">Bölüm</Label>
                  <Popover open={departmentOpen} onOpenChange={setDepartmentOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={departmentOpen}
                        className="w-full justify-between"
                        disabled={!selectedFaculty}
                      >
                        {formData.department || (selectedFaculty ? "Bölüm seçin..." : "Önce fakülte seçin")}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
                      <Command>
                        <CommandInput placeholder="Bölüm ara..." />
                        <CommandList>
                          <CommandEmpty>Bölüm bulunamadı.</CommandEmpty>
                          <CommandGroup>
                            {filteredDepartments.map((dept) => (
                              <CommandItem
                                key={dept.id}
                                value={dept.name}
                                onSelect={() => {
                                  setFormData({
                                    ...formData,
                                    department: formData.department === dept.name ? "" : dept.name,
                                  });
                                  setDepartmentOpen(false);
                                }}
                              >
                                <Check
                                  className={`mr-2 h-4 w-4 ${formData.department === dept.name ? "opacity-100" : "opacity-0"}`}
                                />
                                {dept.name}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="phone">Telefon</Label>
                  <Input
                    id="phone"
                    value={formData.phone}
                    onChange={(e) =>
                      setFormData({ ...formData, phone: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="office_location">Ofis Konumu</Label>
                  <Input
                    id="office_location"
                    value={formData.office_location}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        office_location: e.target.value,
                      })
                    }
                  />
                </div>
                <div className="flex gap-2 pt-4">
                  <Button type="submit" disabled={loading} className="flex-1">
                    {loading ? "Oluşturuluyor..." : "Personel Oluştur"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setCreateDialogOpen(false)}
                    className="flex-1"
                  >
                    İptal
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        {/* Staff Table */}
        <div className="border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort('first_name')}
                    className="h-8 px-2"
                  >
                    Name
                    {sortField === 'first_name' && (
                      sortDirection === 'asc' ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
                    )}
                    {sortField !== 'first_name' && <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort('email')}
                    className="h-8 px-2"
                  >
                    Email
                    {sortField === 'email' && (
                      sortDirection === 'asc' ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
                    )}
                    {sortField !== 'email' && <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort('role')}
                    className="h-8 px-2"
                  >
                    Role
                    {sortField === 'role' && (
                      sortDirection === 'asc' ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
                    )}
                    {sortField !== 'role' && <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort('department')}
                    className="h-8 px-2"
                  >
                    Department
                    {sortField === 'department' && (
                      sortDirection === 'asc' ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
                    )}
                    {sortField !== 'department' && <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />}
                  </Button>
                </TableHead>
                <TableHead>Phone</TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort('office_location')}
                    className="h-8 px-2"
                  >
                    Office
                    {sortField === 'office_location' && (
                      sortDirection === 'asc' ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
                    )}
                    {sortField !== 'office_location' && <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />}
                  </Button>
                </TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && staffList.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    Loading...
                  </TableCell>
                </TableRow>
              ) : staffList.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    No staff members found
                  </TableCell>
                </TableRow>
              ) : (
                sortedStaffList.map((staff) => (
                  <TableRow key={staff.id}>
                    <TableCell className="font-medium">
                      {staff.first_name} {staff.last_name}
                    </TableCell>
                    <TableCell>{staff.email}</TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {ROLE_OPTIONS.find(r => r.value === staff.role)?.label || staff.role}
                      </Badge>
                    </TableCell>
                    <TableCell>{staff.department || "-"}</TableCell>
                    <TableCell>{staff.phone || "-"}</TableCell>
                    <TableCell>{staff.office_location || "-"}</TableCell>
                    <TableCell>
                      <Badge
                        variant={staff.status === "active" ? "default" : "destructive"}
                      >
                        {staff.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right space-x-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openEditDialog(staff)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(staff.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex justify-center gap-2">
            <Button
              variant="outline"
              onClick={() => fetchStaff(page - 1)}
              disabled={page === 1}
            >
              Previous
            </Button>
            <span className="flex items-center px-4">
              Page {page} of {totalPages}
            </span>
            <Button
              variant="outline"
              onClick={() => fetchStaff(page + 1)}
              disabled={page === totalPages}
            >
              Next
            </Button>
          </div>
        )}

        {/* Edit Dialog */}
        <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>
                Edit Staff: {editingStaff?.first_name} {editingStaff?.last_name}
              </DialogTitle>
            </DialogHeader>
            <form onSubmit={handleUpdate} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="edit_email">Email *</Label>
                <Input
                  id="edit_email"
                  type="email"
                  required
                  value={updateFormData.email}
                  onChange={(e) =>
                    setUpdateFormData({ ...updateFormData, email: e.target.value })
                  }
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit_first_name">First Name *</Label>
                  <Input
                    id="edit_first_name"
                    required
                    value={updateFormData.first_name}
                    onChange={(e) =>
                      setUpdateFormData({ ...updateFormData, first_name: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit_last_name">Last Name *</Label>
                  <Input
                    id="edit_last_name"
                    required
                    value={updateFormData.last_name}
                    onChange={(e) =>
                      setUpdateFormData({ ...updateFormData, last_name: e.target.value })
                    }
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_faculty">Fakülte (filtreleme için)</Label>
                <Popover open={editFacultyOpen} onOpenChange={setEditFacultyOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="outline"
                      role="combobox"
                      aria-expanded={editFacultyOpen}
                      className="w-full justify-between"
                    >
                      {editSelectedFaculty || "Fakülte seçin..."}
                      <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
                    <Command>
                      <CommandInput placeholder="Fakülte ara..." />
                      <CommandList>
                        <CommandEmpty>Fakülte bulunamadı.</CommandEmpty>
                        <CommandGroup>
                          {mockFaculties.map((faculty) => (
                            <CommandItem
                              key={faculty.id}
                              value={faculty.name}
                              onSelect={() => {
                                setEditSelectedFaculty(editSelectedFaculty === faculty.name ? "" : faculty.name);
                                setUpdateFormData({ ...updateFormData, department: "" });
                                setEditFacultyOpen(false);
                              }}
                            >
                              <Check
                                className={`mr-2 h-4 w-4 ${editSelectedFaculty === faculty.name ? "opacity-100" : "opacity-0"}`}
                              />
                              {faculty.name}
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_department">Bölüm</Label>
                <Popover open={editDepartmentOpen} onOpenChange={setEditDepartmentOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="outline"
                      role="combobox"
                      aria-expanded={editDepartmentOpen}
                      className="w-full justify-between"
                      disabled={!editSelectedFaculty}
                    >
                      {updateFormData.department || (editSelectedFaculty ? "Bölüm seçin..." : "Önce fakülte seçin")}
                      <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
                    <Command>
                      <CommandInput placeholder="Bölüm ara..." />
                      <CommandList>
                        <CommandEmpty>Bölüm bulunamadı.</CommandEmpty>
                        <CommandGroup>
                          {editFilteredDepartments.map((dept) => (
                            <CommandItem
                              key={dept.id}
                              value={dept.name}
                              onSelect={() => {
                                setUpdateFormData({
                                  ...updateFormData,
                                  department: updateFormData.department === dept.name ? "" : dept.name,
                                });
                                setEditDepartmentOpen(false);
                              }}
                            >
                              <Check
                                className={`mr-2 h-4 w-4 ${updateFormData.department === dept.name ? "opacity-100" : "opacity-0"}`}
                              />
                              {dept.name}
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_phone">Phone</Label>
                <Input
                  id="edit_phone"
                  value={updateFormData.phone}
                  onChange={(e) =>
                    setUpdateFormData({ ...updateFormData, phone: e.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_office_location">Office Location</Label>
                <Input
                  id="edit_office_location"
                  value={updateFormData.office_location}
                  onChange={(e) =>
                    setUpdateFormData({
                      ...updateFormData,
                      office_location: e.target.value,
                    })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_status">Status</Label>
                <select
                  id="edit_status"
                  value={updateFormData.status}
                  onChange={(e) =>
                    setUpdateFormData({ ...updateFormData, status: e.target.value })
                  }
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </div>
              <div className="flex gap-2 pt-4">
                <Button type="submit" disabled={loading} className="flex-1">
                  {loading ? "Updating..." : "Update Staff"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setEditDialogOpen(false)}
                  className="flex-1"
                >
                  Cancel
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}
