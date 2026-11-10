function run(argv) {
  const request = JSON.parse(argv[0]);
  const account = findAccount(request.account);
  const sender = (account.emailAddresses() || [])[0];
  if (!sender) throw new Error('account "' + request.account + '" has no email address');

  const draft = Mail.OutgoingMessage({
    subject: request.subject,
    content: request.body,
    visible: !request.send,
    sender: sender,
  });
  Mail.outgoingMessages.push(draft);
  for (const address of request.to) draft.toRecipients.push(Mail.ToRecipient({ address: address }));
  for (const address of request.cc) draft.ccRecipients.push(Mail.CcRecipient({ address: address }));
  for (const address of request.bcc) draft.bccRecipients.push(Mail.BccRecipient({ address: address }));
  for (const path of request.attachments) draft.attachments.push(Mail.Attachment({ fileName: Path(path) }));

  // Mail loads attachments asynchronously; sending straight away can drop them.
  if (request.attachments.length > 0) delay(2);
  if (request.send) draft.send();
  return JSON.stringify({ sent: request.send, sender: sender });
}
