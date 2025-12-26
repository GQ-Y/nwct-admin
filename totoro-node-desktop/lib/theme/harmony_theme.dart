import 'package:flutter/material.dart';

class HarmonyColors {
  static const primary = Color(0xFF0A59F7);
  static const primaryHover = Color(0xFF3275F9);
  static const primaryActive = Color(0xFF0042D0);

  static const success = Color(0xFF41BA41);
  static const warning = Color(0xFFE8A600);
  static const error = Color(0xFFE84026);

  static const textPrimary = Color(0xFF191919);
  static const textSecondary = Color(0xFF999999);
  static const textTertiary = Color(0xFFB3B3B3);

  static const bgBody = Color(0xFFF1F3F5);
  static const bgSurface = Color(0xFFFFFFFF);
  static const bgInput = Color(0xFFF6F8FA);
  static const bgSidebar = Color(0xB3FFFFFF); // rgba(255,255,255,0.7)

  static const outputBg = Color(0xFF0B1020);
  static const outputFg = Color(0xFFE5E7EB);
}

class HarmonyTheme {
  static ThemeData build() {
    final cs = ColorScheme.fromSeed(
      seedColor: HarmonyColors.primary,
      brightness: Brightness.light,
      primary: HarmonyColors.primary,
      surface: HarmonyColors.bgSurface,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: cs,
      scaffoldBackgroundColor: HarmonyColors.bgBody,
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: false,
      ),
      dividerTheme: DividerThemeData(
        color: Colors.black.withOpacity(0.06),
        thickness: 1,
        space: 1,
      ),
      textTheme:
          const TextTheme(
            titleLarge: TextStyle(fontWeight: FontWeight.w800, fontSize: 18),
            titleMedium: TextStyle(fontWeight: FontWeight.w800, fontSize: 16),
            labelLarge: TextStyle(fontWeight: FontWeight.w700, fontSize: 14),
          ).apply(
            bodyColor: HarmonyColors.textPrimary,
            displayColor: HarmonyColors.textPrimary,
          ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: HarmonyColors.bgInput,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 18,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: const BorderSide(color: HarmonyColors.primary, width: 2),
        ),
        hintStyle: const TextStyle(color: HarmonyColors.textSecondary),
        labelStyle: const TextStyle(color: HarmonyColors.textSecondary),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          ),
          shape: WidgetStatePropertyAll(
            RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
          ),
          backgroundColor: WidgetStateProperty.resolveWith((s) {
            if (s.contains(WidgetState.disabled))
              return HarmonyColors.primary.withOpacity(0.35);
            if (s.contains(WidgetState.pressed))
              return HarmonyColors.primaryActive;
            if (s.contains(WidgetState.hovered))
              return HarmonyColors.primaryHover;
            return HarmonyColors.primary;
          }),
          foregroundColor: const WidgetStatePropertyAll(Colors.white),
          textStyle: const WidgetStatePropertyAll(
            TextStyle(fontWeight: FontWeight.w800, fontSize: 14),
          ),
          elevation: const WidgetStatePropertyAll(0),
          overlayColor: WidgetStatePropertyAll(Colors.white.withOpacity(0.08)),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          ),
          shape: WidgetStatePropertyAll(
            RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
          ),
          side: WidgetStateProperty.resolveWith((s) {
            final base = Colors.black.withOpacity(0.12);
            if (s.contains(WidgetState.hovered))
              return const BorderSide(color: HarmonyColors.primary, width: 1);
            return BorderSide(color: base, width: 1);
          }),
          foregroundColor: WidgetStateProperty.resolveWith((s) {
            if (s.contains(WidgetState.hovered)) return HarmonyColors.primary;
            return HarmonyColors.textPrimary;
          }),
          textStyle: const WidgetStatePropertyAll(
            TextStyle(fontWeight: FontWeight.w800, fontSize: 14),
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          ),
          shape: WidgetStatePropertyAll(
            RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
          ),
          foregroundColor: const WidgetStatePropertyAll(
            HarmonyColors.textSecondary,
          ),
          textStyle: const WidgetStatePropertyAll(
            TextStyle(fontWeight: FontWeight.w700, fontSize: 13),
          ),
          overlayColor: WidgetStatePropertyAll(Colors.black.withOpacity(0.05)),
        ),
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((s) {
          if (s.contains(WidgetState.selected)) return Colors.white;
          return Colors.white;
        }),
        trackColor: WidgetStateProperty.resolveWith((s) {
          if (s.contains(WidgetState.selected)) return HarmonyColors.primary;
          return Colors.black.withOpacity(0.12);
        }),
      ),
    );
  }
}
