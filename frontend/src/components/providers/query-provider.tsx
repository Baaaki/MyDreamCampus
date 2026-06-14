
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 60 * 1000, // 5 dakika boyunca fresh kalır
            gcTime: 10 * 60 * 1000, // 10 dakika cache'de tutulur
            refetchOnWindowFocus: false, // Pencere odaklandığında refetch yapma
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}
