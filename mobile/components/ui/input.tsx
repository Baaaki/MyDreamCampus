import * as React from 'react';
import { TextInput, useColorScheme } from 'react-native';
import { COLORS } from '@/lib/theme';
import { cn } from '@/lib/utils';

type InputProps = React.ComponentProps<typeof TextInput>;

function Input({ className, editable, ...props }: InputProps) {
  const scheme = useColorScheme() ?? 'light';
  return (
    <TextInput
      className={cn(
        'h-14 rounded-2xl border border-input bg-card px-4 text-base text-foreground',
        editable === false && 'opacity-50',
        className
      )}
      placeholderTextColor={COLORS[scheme].mutedForeground}
      editable={editable}
      {...props}
    />
  );
}

export { Input };
