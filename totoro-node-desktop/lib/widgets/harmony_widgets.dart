import 'dart:ui';

import 'package:flutter/material.dart';

import '../theme/harmony_theme.dart';

class HarmonyCard extends StatelessWidget {
  const HarmonyCard({
    super.key,
    this.title,
    this.extra,
    this.glass = false,
    this.padding = const EdgeInsets.all(26),
    this.child,
  });

  final Widget? title;
  final Widget? extra;
  final bool glass;
  final EdgeInsets padding;
  final Widget? child;

  @override
  Widget build(BuildContext context) {
    final content = Container(
      decoration: BoxDecoration(
        color: glass
            ? HarmonyColors.bgSurface.withOpacity(0.75)
            : HarmonyColors.bgSurface,
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: Colors.white.withOpacity(0.80)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.04),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      padding: padding,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (title != null) ...[
            Row(
              children: [
                DefaultTextStyle.merge(
                  style: const TextStyle(
                    fontWeight: FontWeight.w800,
                    fontSize: 16,
                  ),
                  child: title!,
                ),
                const Spacer(),
                if (extra != null) extra!,
              ],
            ),
            const SizedBox(height: 18),
          ],
          if (child != null) child!,
        ],
      ),
    );

    if (!glass) return content;
    // macOS/desktop 在某些时序下 BackdropFilter + ClipRRect 可能触发 “RenderBox was not laid out”.
    // 这里先保留玻璃拟态的半透明效果，但不做 blur，保证稳定性。
    return content;
  }
}

enum HarmonyButtonVariant { primary, outline, ghost }

class HarmonyButton extends StatelessWidget {
  const HarmonyButton({
    super.key,
    required this.child,
    this.onPressed,
    this.variant = HarmonyButtonVariant.primary,
  });

  final Widget child;
  final VoidCallback? onPressed;
  final HarmonyButtonVariant variant;

  @override
  Widget build(BuildContext context) {
    return switch (variant) {
      HarmonyButtonVariant.primary => ElevatedButton(
        onPressed: onPressed,
        child: child,
      ),
      HarmonyButtonVariant.outline => OutlinedButton(
        onPressed: onPressed,
        child: child,
      ),
      HarmonyButtonVariant.ghost => TextButton(
        onPressed: onPressed,
        child: child,
      ),
    };
  }
}

class HarmonyField extends StatefulWidget {
  const HarmonyField({
    super.key,
    required this.controller,
    required this.label,
    this.hintText,
    this.keyboardType,
    this.obscureText = false,
    this.maxLines = 1,
  });

  final TextEditingController controller;
  final String label;
  final String? hintText;
  final TextInputType? keyboardType;
  final bool obscureText;
  final int maxLines;

  @override
  State<HarmonyField> createState() => _HarmonyFieldState();
}

class _HarmonyFieldState extends State<HarmonyField> {
  late final FocusNode _focus = FocusNode();

  @override
  void initState() {
    super.initState();
    _focus.addListener(() {
      if (!mounted) return;
      // 避免在 pointer/mouse tracker 更新阶段同步 setState 导致断言重入
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        setState(() {});
      });
    });
  }

  @override
  void dispose() {
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final focused = _focus.hasFocus;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 220),
      curve: Curves.easeOutCubic,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        boxShadow: focused
            ? [
                BoxShadow(
                  color: HarmonyColors.primary.withOpacity(0.10),
                  blurRadius: 24,
                  spreadRadius: 1,
                ),
              ]
            : null,
      ),
      child: TextField(
        focusNode: _focus,
        controller: widget.controller,
        keyboardType: widget.keyboardType,
        obscureText: widget.obscureText,
        maxLines: widget.maxLines,
        decoration: InputDecoration(
          labelText: widget.label,
          hintText: widget.hintText,
        ),
      ),
    );
  }
}

class OutputPanel extends StatelessWidget {
  const OutputPanel({super.key, required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: HarmonyColors.outputBg,
        borderRadius: BorderRadius.circular(16),
      ),
      padding: const EdgeInsets.all(12),
      child: SelectableText(
        text,
        style: const TextStyle(
          fontFamily: 'monospace',
          fontSize: 12,
          color: HarmonyColors.outputFg,
        ),
      ),
    );
  }
}

void showToast(BuildContext context, String message) {
  final m = ScaffoldMessenger.of(context);
  m.clearSnackBars();
  m.showSnackBar(
    SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
      backgroundColor: Colors.black.withOpacity(0.88),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      duration: const Duration(seconds: 2),
    ),
  );
}
