import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:cholopay_wallet/main.dart';
import 'package:cholopay_wallet/obligations_client.dart';

void main() {
  testWidgets('muestra obligaciones participant-scoped y permite iniciar pago sintético', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp(
      obligationsClient: SyntheticObligationsClient(obligations: syntheticObligations),
    ));

    await tester.pumpAndSettle();

    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();

    expect(find.text('Obligaciones sintéticas'), findsOneWidget);
    expect(find.text('Luz sintética'), findsOneWidget);
    expect(find.text('Matrícula sintética'), findsOneWidget);
    expect(find.textContaining('Participante COSTA'), findsNWidgets(2));

    await tester.tap(find.byKey(const Key('pay-obligation-bill-light-001')));
    await tester.pumpAndSettle();

    expect(find.text('Pago iniciado: PENDING'), findsOneWidget);
  });

  testWidgets('muestra error explícito al fallar la consulta de obligaciones', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(CholoPayWalletApp(
      obligationsClient: SyntheticObligationsClient(failure: Exception('gateway caído')),
    ));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();

    expect(find.text('No se pudieron consultar las obligaciones'), findsOneWidget);
    expect(find.text('gateway caído'), findsOneWidget);
  });

  testWidgets('muestra estado vacío explícito sin inventar obligaciones', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp(
      obligationsClient: SyntheticObligationsClient(obligations: <Obligation>[]),
    ));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();

    expect(find.text('No hay obligaciones abiertas'), findsOneWidget);
    expect(find.text('Luz sintética'), findsNothing);
  });

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
    expect(find.text('Idempotencia: wallet-confirm-000000000000001'), findsOneWidget);
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
