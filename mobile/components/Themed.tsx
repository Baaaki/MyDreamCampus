import { Text as DefaultText, View as DefaultView } from 'react-native';
import { useTheme } from 'react-native-paper';

type ThemeProps = {
  lightColor?: string;
  darkColor?: string;
};

export type TextProps = ThemeProps & DefaultText['props'];
export type ViewProps = ThemeProps & DefaultView['props'];

export function Text(props: TextProps) {
  const { style, lightColor, darkColor, ...otherProps } = props;
  const { colors } = useTheme();

  return <DefaultText style={[{ color: colors.onSurface }, style]} {...otherProps} />;
}

export function View(props: ViewProps) {
  const { style, lightColor, darkColor, ...otherProps } = props;
  const { colors } = useTheme();

  return <DefaultView style={[{ backgroundColor: colors.background }, style]} {...otherProps} />;
}
