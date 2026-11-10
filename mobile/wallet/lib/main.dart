import 'package:flutter/material.dart';

void main() {
  runApp(const CholoPayWalletApp());
}

class CholoPayWalletApp extends StatelessWidget {
  const CholoPayWalletApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'CholoPay Wallet',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff185c50)),
        useMaterial3: true,
      ),
      home: const WalletShell(),
    );
  }
}

class WalletSession {
  const WalletSession({
    required this.sessionId,
    required this.participantId,
    required this.accountIds,
    required this.scopes,
  });

  final String sessionId;
  final String participantId;
  final List<String> accountIds;
  final List<String> scopes;
}

class WalletAccount {
  const WalletAccount({
    required this.id,
    required this.label,
    required this.currency,
    required this.balanceMinor,
  });

  final String id;
  final String label;
  final String currency;
  final int balanceMinor;
}

class CholoKey {
  const CholoKey({
    required this.key,
    required this.participantId,
    required this.displayName,
  });

  final String key;
  final String participantId;
  final String displayName;
}

class CholoQrPayload {
  const CholoQrPayload({
    required this.payloadId,
    required this.mode,
    required this.destinationKey,
    required this.amountMinor,
    required this.currency,
  });

  final String payloadId;
  final String mode;
  final String destinationKey;
  final int? amountMinor;
  final String currency;
}

class TransferFixture {
  const TransferFixture({
    required this.transferId,
    required this.idempotencyKey,
    required this.state,
    required this.amountMinor,
    required this.currency,
  });

  final String transferId;
  final String idempotencyKey;
  final String state;
  final int amountMinor;
  final String currency;
}

class MovementFixture {
  const MovementFixture({
    required this.transactionId,
    required this.direction,
    required this.state,
    required this.amountMinor,
  });

  final String transactionId;
  final String direction;
  final String state;
  final int amountMinor;
}

class SyntheticConfirmationResult {
  const SyntheticConfirmationResult({required this.wasCreated});

  final bool wasCreated;
}

class SyntheticConfirmationStore {
  TransferFixture? _confirmedTransfer;

  SyntheticConfirmationResult confirm(TransferFixture transfer) {
    if (_confirmedTransfer != null &&
        _confirmedTransfer!.idempotencyKey == transfer.idempotencyKey) {
      return const SyntheticConfirmationResult(wasCreated: false);
    }

    _confirmedTransfer = transfer;
    return const SyntheticConfirmationResult(wasCreated: true);
  }
}

const sessionFixture = WalletSession(
  sessionId: 'sess-0001',
  participantId: 'COSTA',
  accountIds: ['account-costa-001', 'account-costa-002'],
  scopes: ['accounts:read', 'transfers:read', 'keys:read', 'qr:read'],
);

const accountFixtures = [
  WalletAccount(
    id: 'account-costa-001',
    label: 'Costa Bank · USD account',
    currency: 'USD',
    balanceMinor: 125000,
  ),
  WalletAccount(
    id: 'account-costa-002',
    label: 'Costa Savings · USD account',
    currency: 'USD',
    balanceMinor: 450000,
  ),
];

const receiverKeyFixture = CholoKey(
  key: 'ck_demo_receiver_001',
  participantId: 'SIERRA',
  displayName: 'Synthetic Receiver',
);

const qrFixture = CholoQrPayload(
  payloadId: 'qr-payload-001',
  mode: 'static',
  destinationKey: 'ck_demo_receiver_001',
  amountMinor: null,
  currency: 'USD',
);

const transferFixture = TransferFixture(
  transferId: '00000000-0000-0000-0000-000000000010',
  idempotencyKey: 'wallet-confirm-000000000000001',
  state: 'READY_FOR_SETTLEMENT',
  amountMinor: 2500,
  currency: 'USD',
);

const movementFixtures = [
  MovementFixture(
    transactionId: 'txn-demo-001',
    direction: 'IN',
    state: 'COMPLETED',
    amountMinor: 10000,
  ),
  MovementFixture(
    transactionId: 'txn-demo-002',
    direction: 'OUT',
    state: 'SETTLED',
    amountMinor: 2500,
  ),
];

class WalletShell extends StatefulWidget {
  const WalletShell({super.key});

  @override
  State<WalletShell> createState() => _WalletShellState();
}

class _WalletShellState extends State<WalletShell> {
  WalletAccount selectedAccount = accountFixtures.first;
  final SyntheticConfirmationStore _confirmationStore =
      SyntheticConfirmationStore();
  String confirmationState = 'NOT CONFIRMED';
  bool wasDuplicateReplay = false;
  int transfersCreated = 0;

  void _confirmTransfer() {
    final result = _confirmationStore.confirm(transferFixture);
    setState(() {
      confirmationState = 'CONFIRMED';
      wasDuplicateReplay = !result.wasCreated;
      if (result.wasCreated) transfersCreated += 1;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('CholoPay Wallet'),
        actions: const [Padding(padding: EdgeInsets.all(16), child: Icon(Icons.lock_outline))],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 32),
        children: [
          const _ReadOnlyBanner(),
          const SizedBox(height: 16),
          DropdownButtonFormField<WalletAccount>(
            key: const Key('account-selector'),
            value: selectedAccount,
            decoration: const InputDecoration(
              labelText: 'Selected account',
              border: OutlineInputBorder(),
            ),
            items: [
              for (final account in accountFixtures)
                DropdownMenuItem(value: account, child: Text(account.label)),
            ],
            onChanged: (account) {
              if (account != null) setState(() => selectedAccount = account);
            },
          ),
          const SizedBox(height: 16),
          _BalanceCard(account: selectedAccount),
          const SizedBox(height: 16),
          const _QrCard(payload: qrFixture, receiver: receiverKeyFixture),
          const SizedBox(height: 16),
          _TransferCard(
            transfer: transferFixture,
            payer: selectedAccount,
            payee: receiverKeyFixture,
            confirmationState: confirmationState,
            wasDuplicateReplay: wasDuplicateReplay,
            transfersCreated: transfersCreated,
            onConfirm: _confirmTransfer,
          ),
          const SizedBox(height: 16),
          const _MovementsCard(movements: movementFixtures),
        ],
      ),
    );
  }
}

class _ReadOnlyBanner extends StatelessWidget {
  const _ReadOnlyBanner();

  @override
  Widget build(BuildContext context) => Card(
        color: Theme.of(context).colorScheme.secondaryContainer,
        child: const ListTile(
          leading: Icon(Icons.visibility_outlined),
          title: Text('Read-only demo'),
          subtitle: Text('Local synthetic fixtures · no backend or database access'),
        ),
      );
}

class _BalanceCard extends StatelessWidget {
  const _BalanceCard({required this.account});
  final WalletAccount account;

  @override
  Widget build(BuildContext context) => Card(
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Available balance'),
            const SizedBox(height: 6),
            Text(
              '\$${(account.balanceMinor / 100).toStringAsFixed(2)} ${account.currency}',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 8),
            Text('Session ${sessionFixture.sessionId} · ${sessionFixture.participantId}'),
          ]),
        ),
      );
}

class _QrCard extends StatelessWidget {
  const _QrCard({required this.payload, required this.receiver});
  final CholoQrPayload payload;
  final CholoKey receiver;

  @override
  Widget build(BuildContext context) => Card(
        child: ListTile(
          leading: const Icon(Icons.qr_code_2, size: 36),
          title: const Text('CholoPay QR'),
          subtitle: Text('${payload.mode.toUpperCase()} · ${receiver.displayName} · ${receiver.key}'),
          trailing: const Text('Synthetic'),
        ),
      );
}

class _TransferCard extends StatelessWidget {
  const _TransferCard({
    required this.transfer,
    required this.payer,
    required this.payee,
    required this.confirmationState,
    required this.wasDuplicateReplay,
    required this.transfersCreated,
    required this.onConfirm,
  });

  final TransferFixture transfer;
  final WalletAccount payer;
  final CholoKey payee;
  final String confirmationState;
  final bool wasDuplicateReplay;
  final int transfersCreated;
  final VoidCallback onConfirm;

  @override
  Widget build(BuildContext context) => Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Synthetic confirmation',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Text('Payer: ${payer.label}'),
              Text('Payee: ${payee.displayName}'),
              Text('Amount: \$${(transfer.amountMinor / 100).toStringAsFixed(2)} ${transfer.currency}'),
              const SizedBox(height: 8),
              Text('Confirmation: $confirmationState'),
              if (wasDuplicateReplay)
                const Text('Duplicate replay: REUSED_EXISTING_TRANSFER'),
              Text('Transfer ID: ${transfer.transferId}'),
              Text('Idempotency: ${transfer.idempotencyKey}'),
              Text('Transfers created: $transfersCreated'),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton(
                  key: const Key('confirm-transfer'),
                  onPressed: onConfirm,
                  child: const Text('Confirm synthetic transfer'),
                ),
              ),
              const SizedBox(height: 4),
              const Text('Read-only local result; no ledger or backend access.'),
            ],
          ),
        ),
      );
}

class _MovementsCard extends StatelessWidget {
  const _MovementsCard({required this.movements});
  final List<MovementFixture> movements;

  @override
  Widget build(BuildContext context) => Card(
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Text('Recent movements', style: TextStyle(fontWeight: FontWeight.bold)),
          ),
          for (final movement in movements)
            ListTile(
              key: ValueKey(movement.transactionId),
              leading: Icon(movement.direction == 'IN' ? Icons.south_west : Icons.north_east),
              title: Text(movement.transactionId),
              subtitle: Text('${movement.direction} · ${movement.state}'),
              trailing: Text('\$${(movement.amountMinor / 100).toStringAsFixed(2)}'),
            ),
        ]),
      );
}
