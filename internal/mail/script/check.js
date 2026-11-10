function main(request) {
  if (request.account === 'local') Mail.checkForNewMail();
  else Mail.checkForNewMail({ for: findAccount(request.account) });
  return {};
}
