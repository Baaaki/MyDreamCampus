'use client';

import Link from 'next/link';
import {
  Users,
  GraduationCap,
  BookOpen,
  ClipboardCheck,
  FileCheck,
  BarChart3,
  UtensilsCrossed,
  TrendingUp,
  UserCheck,
  Calendar,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

const stats = [
  {
    title: 'Toplam Öğrenci',
    value: '12,543',
    change: '+2.5%',
    changeType: 'positive' as const,
    icon: GraduationCap,
    color: 'bg-blue-500',
  },
  {
    title: 'Toplam Personel',
    value: '1,234',
    change: '+1.2%',
    changeType: 'positive' as const,
    icon: Users,
    color: 'bg-green-500',
  },
  {
    title: 'Aktif Ders',
    value: '456',
    change: '+5 yeni',
    changeType: 'positive' as const,
    icon: BookOpen,
    color: 'bg-purple-500',
  },
  {
    title: 'Bugünkü Yoklama',
    value: '89%',
    change: '+3%',
    changeType: 'positive' as const,
    icon: UserCheck,
    color: 'bg-orange-500',
  },
];

const quickLinks = [
  {
    title: 'Personel Yönetimi',
    description: 'Personel ekle, düzenle ve yönet',
    href: '/staff',
    icon: Users,
    color: 'from-green-500 to-emerald-600',
  },
  {
    title: 'Öğrenci Yönetimi',
    description: 'Öğrenci kayıtları ve danışman atamaları',
    href: '/students',
    icon: GraduationCap,
    color: 'from-blue-500 to-cyan-600',
  },
  {
    title: 'Ders Kataloğu',
    description: 'Ders ekle ve müfredatı yönet',
    href: '/catalog',
    icon: BookOpen,
    color: 'from-purple-500 to-pink-600',
  },
  {
    title: 'Ders Kayıt',
    description: 'Ders kayıt işlemlerini yönet',
    href: '/enrollment',
    icon: ClipboardCheck,
    color: 'from-pink-500 to-rose-600',
  },
  {
    title: 'Yoklama Sistemi',
    description: 'Yoklama takibi ve raporları',
    href: '/attendance/teacher',
    icon: FileCheck,
    color: 'from-red-500 to-orange-600',
  },
  {
    title: 'Not Yönetimi',
    description: 'Not girişi ve transkript',
    href: '/grades/teacher',
    icon: BarChart3,
    color: 'from-indigo-500 to-blue-600',
  },
  {
    title: 'Yemekhane',
    description: 'Yemekhane rezervasyon yönetimi',
    href: '/meal/admin',
    icon: UtensilsCrossed,
    color: 'from-orange-500 to-amber-600',
  },
  {
    title: 'Oturumlar',
    description: 'Aktif oturum yönetimi',
    href: '/auth/sessions',
    icon: Calendar,
    color: 'from-gray-500 to-slate-600',
  },
];

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Hoş geldiniz! Kampüs yönetim sistemine genel bakış.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Card key={stat.title} className="dark:bg-gray-900 dark:border-gray-800">
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-500 dark:text-gray-400">{stat.title}</p>
                    <p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
                      {stat.value}
                    </p>
                    <p className={`text-sm mt-1 ${
                      stat.changeType === 'positive' ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {stat.change}
                    </p>
                  </div>
                  <div className={`p-3 rounded-lg ${stat.color}`}>
                    <Icon className="h-6 w-6 text-white" />
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Quick Links */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Hızlı Erişim
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {quickLinks.map((link) => {
            const Icon = link.icon;
            return (
              <Link
                key={link.href}
                href={link.href}
                className="group block"
              >
                <Card className="h-full transition-all hover:shadow-lg hover:-translate-y-1 dark:bg-gray-900 dark:border-gray-800 dark:hover:border-gray-700">
                  <CardContent className="p-6">
                    <div className={`inline-flex p-3 rounded-lg bg-gradient-to-br ${link.color} mb-4`}>
                      <Icon className="h-6 w-6 text-white" />
                    </div>
                    <h3 className="font-semibold text-gray-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                      {link.title}
                    </h3>
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                      {link.description}
                    </p>
                  </CardContent>
                </Card>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="text-lg dark:text-white">Son Aktiviteler</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { action: 'Yeni öğrenci kaydı', user: 'Ahmet Yılmaz', time: '5 dk önce' },
                { action: 'Ders programı güncellendi', user: 'Prof. Dr. Mehmet Demir', time: '15 dk önce' },
                { action: 'Yoklama alındı', user: 'BIL 101', time: '30 dk önce' },
                { action: 'Not girişi yapıldı', user: 'FIZ 102', time: '1 saat önce' },
                { action: 'Yemek rezervasyonu', user: '150 öğrenci', time: '2 saat önce' },
              ].map((activity, index) => (
                <div key={index} className="flex items-center justify-between py-2 border-b border-gray-100 dark:border-gray-800 last:border-0">
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-white">{activity.action}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{activity.user}</p>
                  </div>
                  <span className="text-xs text-gray-400">{activity.time}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="text-lg dark:text-white">Sistem Durumu</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { service: 'Auth Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Staff Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Student Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Course Catalog Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Enrollment Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Attendance Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Grades Service', status: 'Aktif', color: 'bg-green-500' },
                { service: 'Meal Service', status: 'Aktif', color: 'bg-green-500' },
              ].map((service, index) => (
                <div key={index} className="flex items-center justify-between py-2 border-b border-gray-100 dark:border-gray-800 last:border-0">
                  <span className="text-sm text-gray-700 dark:text-gray-300">{service.service}</span>
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${service.color}`}></div>
                    <span className="text-sm text-gray-500 dark:text-gray-400">{service.status}</span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
