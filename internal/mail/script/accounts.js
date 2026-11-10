function main() {
  let names;
  try {
    names = Mail.accounts.name() || [];
  } catch (error) {
    if (error.errorNumber === -1728) return [];
    throw error;
  }
  if (names.length === 0) return [];
  const emails = Mail.accounts.emailAddresses();
  const kinds = Mail.accounts.accountType();
  const enabled = Mail.accounts.enabled();
  return names.map((name, position) => ({
    name: name,
    addresses: emails[position] || [],
    kind: String(kinds[position]),
    enabled: Boolean(enabled[position]),
  }));
}
