function run() {
  const accounts = Mail.accounts().map((account) => ({
    name: account.name(),
    addresses: account.emailAddresses() || [],
    kind: String(account.accountType()),
    enabled: account.enabled(),
  }));
  return JSON.stringify(accounts);
}
