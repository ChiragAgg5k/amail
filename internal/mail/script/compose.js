function main(request) {
  if (request.account === 'local') throw refuse('local mailboxes cannot send mail');
  const account = findAccount(request.account);
  const owned = account.emailAddresses() || [];
  const sender = request.from || owned[0];
  if (!sender) throw missing('account', 'account "' + request.account + '" has no email address');
  if (!owned.some(function (address) { return sameAddress(address, sender); })) {
    throw missing('account', 'account "' + request.account + '" has no address ' + sender);
  }

  let signature = null;
  if (request.signature) {
    signature = Mail.signatures.byName(request.signature);
    if (!signature.exists()) throw missing('signature', 'no signature named "' + request.signature + '"');
  }

  const draft = Mail.OutgoingMessage({
    subject: request.subject,
    content: request.body,
    visible: !request.send,
    sender: sender,
  });
  Mail.outgoingMessages.push(draft);
  try {
    for (const address of request.to) draft.toRecipients.push(Mail.ToRecipient({ address: address }));
    for (const address of request.cc) draft.ccRecipients.push(Mail.CcRecipient({ address: address }));
    for (const address of request.bcc) draft.bccRecipients.push(Mail.BccRecipient({ address: address }));
    for (const path of request.attachments) draft.attachments.push(Mail.Attachment({ fileName: Path(path) }));
    if (signature) draft.messageSignature = signature;

    const actual = Mail.extractAddressFrom(draft.sender() || '');
    if (!sameAddress(actual, sender)) {
      throw refuse('Mail set the sender to ' + (actual || 'nothing') + ' instead of ' + sender + '; nothing was sent');
    }
  } catch (error) {
    draft.delete();
    throw error;
  }

  // Mail loads attachments asynchronously; sending straight away can drop them.
  if (request.attachments.length > 0) delay(2);
  if (request.send && !draft.send()) throw new Error('Mail did not send the message');
  return { sent: request.send, sender: sender };
}

function sameAddress(left, right) {
  return String(left).toLowerCase() === String(right).toLowerCase();
}
