import { redirect } from 'next/navigation';

export default function HomePage() {
  // Ana sayfa doğrudan login'e yönlendirir
  redirect('/auth/login');
}