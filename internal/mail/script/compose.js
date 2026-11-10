// The sender is set to the request value as given, so a display name in
// `Name <address>` survives; every comparison uses the bare address only.
function main(request) {
  if (request.account === 'local') throw refuse('local mailboxes cannot send mail');
  const account = findAccount(request.account);
  const owned = account.emailAddresses() || [];
  const sender = request.from || owned[0];
  if (!sender) throw missing('account', 'account "' + request.account + '" has no email address');
  const wanted = bareAddress(sender);
  if (!owned.some(function (address) { return sameAddress(address, wanted); })) {
    throw missing('account', 'account "' + request.account + '" has no address ' + wanted);
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
    // The dictionary declares `attachment` as an element of the rich text content,
    // not of the outgoing message, so files are added to the content. Needs
    // confirming end-to-end; the AppleScript equivalent is
    // `make new attachment with properties {file name:...} at after last paragraph`.
    for (const path of request.attachments) draft.content.attachments.push(Mail.Attachment({ fileName: Path(path) }));
    if (signature) draft.messageSignature = signature;

    const actual = bareAddress(draft.sender() || '');
    if (!sameAddress(actual, wanted)) {
      throw refuse('Mail set the sender to ' + (actual || 'nothing') + ' instead of ' + wanted + '; nothing was sent');
    }

    // Mail loads attachments asynchronously; sending straight away can drop them.
    if (request.attachments.length > 0) delay(2);
    if (request.send && !draft.send()) throw new Error('Mail did not send the message');
  } catch (error) {
    // Outgoing messages respond only to save and close, not delete.
    try {
      Mail.close(draft, { saving: 'no' });
    } catch (ignored) {}
    throw error;
  }
  return { sent: request.send, sender: sender };
}

function bareAddress(value) {
  const text = String(value || '');
  if (text === '') return '';
  return Mail.extractAddressFrom(text) || text;
}

function sameAddress(left, right) {
  return String(left).trim().toLowerCase() === String(right).trim().toLowerCase();
}
