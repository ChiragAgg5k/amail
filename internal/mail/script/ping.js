function main(request) {
  return { version: Mail.version(), accounts: Mail.accounts.length };
}
