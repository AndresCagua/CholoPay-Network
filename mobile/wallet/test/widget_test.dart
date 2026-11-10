import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:cholopay_wallet/main.dart';

void main() {
  testWidgets('renders the selected synthetic account and read-only sections', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());

    expect(find.text('CholoPay Wallet'), findsOneWidget);
    expect(find.text('Costa Bank · USD account'), findsOneWidget);
    expect(find.text('CholoPay QR'), findsOneWidget);
    expect(find.text('Read-only demo'), findsOneWidget);
    await tester.drag(find.byType(ListView), const Offset(0, -700));
    await tester.pumpAndSettle();
    expect(find.text('Recent movements'), findsOneWidget);
  });

  testWidgets('can select another local account fixture', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());

    await tester.tap(find.byKey(const Key('account-selector')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Costa Savings · USD account').last);
    await tester.pumpAndSettle();

    expect(find.text('Costa Savings · USD account'), findsOneWidget);
  });

  testWidgets('confirms the synthetic payer and payee transfer once', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const CholoPayWalletApp());
    await tester.drag(find.byType(ListView), const Offset(0, -350));
    await tester.pumpAndSettle();

    expect(find.text('Confirmation: NOT CONFIRMED'), findsOneWidget);
    expect(find.text('Payer: Costa Bank · USD account'), findsOneWidget);
    expect(find.text('Payee: Synthetic Receiver'), findsOneWidget);

    await tester.tap(find.byKey(const Key('confirm-transfer')));
    await tester.pumpAndSettle();

    expect(find.text('Confirmation: CONFIRMED'), findsOneWidget);
    expect(find.text('Transfer ID: 00000000-0000-0000-0000-000000000010'), findsOneWidget);
    expect(find.text('Transfers created: 1'), findsOneWidget);
    expect(find.text('Idempotency: wallet-confirm-000000000000001'), findsOneWidget);
  });

  testWidgets('replays duplicate confirmation without creating another transfer', (
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

    expect(find.text('Duplicate replay: REUSED_EXISTING_TRANSFER'), findsOneWidget);
    expect(find.text('Confirmation: CONFIRMED'), findsOneWidget);
    expect(find.text('Transfers created: 1'), findsOneWidget);
    expect(find.text('Transfer ID: 00000000-0000-0000-0000-000000000010'), findsOneWidget);
  });
}
