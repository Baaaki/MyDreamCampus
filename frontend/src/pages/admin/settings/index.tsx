
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useTheme } from '@/components/providers/theme-provider';
import { useNavigate } from 'react-router';
import { Moon, Sun, Key, Shield, Bell, User } from 'lucide-react';

export default function SettingsPage() {
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Ayarlar</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Hesap ve sistem ayarlarınızı yönetin
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Tema Ayarları */}
        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 dark:text-white">
              {theme === 'light' ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
              Tema
            </CardTitle>
            <CardDescription>Görünüm tercihlerinizi ayarlayın</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium dark:text-white">
                  {theme === 'light' ? 'Açık Tema' : 'Koyu Tema'}
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {theme === 'light' ? 'Gündüz modu aktif' : 'Gece modu aktif'}
                </p>
              </div>
              <Button variant="outline" onClick={toggleTheme}>
                {theme === 'light' ? (
                  <>
                    <Moon className="h-4 w-4 mr-2" />
                    Koyu Tema
                  </>
                ) : (
                  <>
                    <Sun className="h-4 w-4 mr-2" />
                    Açık Tema
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Güvenlik */}
        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 dark:text-white">
              <Shield className="h-5 w-5" />
              Güvenlik
            </CardTitle>
            <CardDescription>Hesap güvenlik ayarları</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Button
              variant="outline"
              className="w-full justify-start"
              onClick={() => navigate('/auth/change-password')}
            >
              <Key className="h-4 w-4 mr-2" />
              Şifre Değiştir
            </Button>
            <Button
              variant="outline"
              className="w-full justify-start"
              onClick={() => navigate('/auth/sessions')}
            >
              <User className="h-4 w-4 mr-2" />
              Aktif Oturumlar
            </Button>
          </CardContent>
        </Card>

        {/* Bildirimler */}
        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 dark:text-white">
              <Bell className="h-5 w-5" />
              Bildirimler
            </CardTitle>
            <CardDescription>Bildirim tercihlerinizi yönetin</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Bildirim ayarları yakında eklenecek.
            </p>
          </CardContent>
        </Card>

        {/* Profil */}
        <Card className="dark:bg-gray-900 dark:border-gray-800">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 dark:text-white">
              <User className="h-5 w-5" />
              Profil
            </CardTitle>
            <CardDescription>Profil bilgilerinizi düzenleyin</CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              className="w-full justify-start"
              onClick={() => navigate('/staff/profile')}
            >
              <User className="h-4 w-4 mr-2" />
              Profili Görüntüle
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
