# Bridge for Broom Ledger Token to Solana  

A light Broom Ledger node that watches the broom ledger and mints broomledger on the sol chain.

The bridge node will also transfer token back from the SOL network to the Broom network.

## Quick Start (Broom -> SOL)

Send a tansaction using any wallet to the Bridge token address

### Address

Send the transaction to the Bridge node address.

```bash
5XGQBMBBPuzaA3nBbLbFhyytCT8dozt5U9WTubuFxpac
```

### Note

Set the note in your transaction to `your` SOL address. (Not your Associated Transaction Address)

Example:

```bash
5EzYAKuTXtPadcmozKJW3GfBZ244K2fk79SfLrnzkwcu
```

### Fees and Minimums

The minimum transaction amount is `10,000` base units OR `0.1` broom token.

Creating a new ATA for the sol account on the ledger is costly and will require rent fees. The initial fees incurred will be `10%` of the Broom transaction. Every subsequent transaction will be charged `0.1%`. This covers small lamports fees on the SOL network.

### Transaction Finalization

It can take up to 20 min to move token across the bridge. If there are any issues and you never received token please reach out to: `support@broomledger.com`

## Quick Start (SOL -> Broom)

Send a transaction for SPL token `EJtfMsN3qfh8QJJfpEmWVxW43MPH522xtXBfvJNA9Bdk` to the bridge account address:

```bash
5EzYAKuTXtPadcmozKJW3GfBZ244K2fk79SfLrnzkwcu
```
