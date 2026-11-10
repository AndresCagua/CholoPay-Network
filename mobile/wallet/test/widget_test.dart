import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:cholopay_wallet/main.dart';

void main() {
  testWidgets('renderiza la cuenta sintética seleccionada y las secciones de solo lectura', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());

    expect(find.text('Billetera CholoPay'), findsOneWidget);
    expect(find.text('Banco Costa · cuenta USD'), findsOneWidget);
    expect(find.text('CholoPay QR'), findsOneWidget);
    expect(find.text('Demostración de solo lectura'), findsOneWidget);
    await tester.drag(find.byType(ListView), const Offset(0, -700));
    await tester.pumpAndSettle();
    expect(find.text('Movimientos recientes'), findsOneWidget);
  });

  testWidgets('permite seleccionar otro dato sintético de cuenta local', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());

    await tester.tap(find.byKey(const Key('account-selector')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Ahorros Costa · cuenta USD').last);
    await tester.pumpAndSettle();

    expect(find.text('Ahorros Costa · cuenta USD'), findsOneWidget);
  });

  testWidgets('confirma una vez la transferencia del pagador y beneficiario sintéticos', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());
    await tester.drag(find.byType(ListView), const Offset(0, -350));
    await tester.pumpAndSettle();

    expect(find.text('Confirmación: NOT CONFIRMED'), findsOneWidget);
    expect(find.text('Pagador: Banco Costa · cuenta USD'), findsOneWidget);
    expect(find.text('Beneficiario: Receptor sintético'), findsOneWidget);

    await tester.tap(find.byKey(const Key('confirm-transfer')));
    await tester.pumpAndSettle();

    expect(find.text('Confirmación: CONFIRMED'), findsOneWidget);
    expect(find.text('ID de transferencia: 00000000-0000-0000-0000-000000000010'), findsOneWidget);
    expect(find.text('Transferencias creadas: 1'), findsOneWidget);
    expect(find.text('Idempotency: wallet-confirm-000000000000001'), findsOneWidget);
  });

  testWidgets('reproduce la confirmación duplicada sin crear otra transferencia', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());
    await tester.drag(find.byType(ListView), const Offset(0, -350));
    await tester.pumpAndSettle();

    final confirmButton = find.byKey(const Key('confirm-transfer'));
    await tester.tap(confirmButton);
    await tester.pumpAndSettle();
    await tester.tap(confirmButton);
    await tester.pumpAndSettle();

    expect(find.text('Repetición duplicada: REUSED_EXISTING_TRANSFER'), findsOneWidget);
    expect(find.text('Confirmación: CONFIRMED'), findsOneWidget);
    expect(find.text('Transferencias creadas: 1'), findsOneWidget);
    expect(find.text('ID de transferencia: 00000000-0000-0000-0000-000000000010'), findsOneWidget);
  });
}
