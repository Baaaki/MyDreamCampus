import * as Slot from '@rn-primitives/slot';
import * as React from 'react';
import { Text as RNText } from 'react-native';
import { cn } from '@/lib/utils';

// Buton gibi kapsayicilarin icindeki Text'e varyant stili aktarmak icin
// (react-native-reusables pattern'i).
const TextClassContext = React.createContext<string | undefined>(undefined);

type TextProps = React.ComponentProps<typeof RNText> & {
  asChild?: boolean;
};

function Text({ className, asChild = false, ...props }: TextProps) {
  const textClass = React.useContext(TextClassContext);
  const Component = asChild ? Slot.Text : RNText;
  return (
    <Component
      className={cn('text-base text-foreground', textClass, className)}
      {...props}
    />
  );
}

export { Text, TextClassContext };
