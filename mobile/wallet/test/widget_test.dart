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
}
