function main(request) {
  const found = findMessage(request);
  const message = found.message;
  const sent = optional(() => message.dateSent());
  const received = optional(() => message.dateReceived());
  return {
    id: message.id(),
    messageId: optional(() => message.messageId()) || '',
    mailbox: found.mailbox,
    subject: optional(() => message.subject()) || '',
    sender: optional(() => message.sender()) || '',
    replyTo: optional(() => message.replyTo()) || '',
    to: addresses(message.toRecipients),
    cc: addresses(message.ccRecipients),
    sent: sent ? sent.toISOString() : null,
    received: received ? received.toISOString() : null,
    read: message.readStatus(),
    flagged: message.flaggedStatus(),
    content: optional(() => message.content()) || '',
    attachments: message.mailAttachments().map((attachment) => ({
      name: attachment.name(),
      size: optional(() => attachment.fileSize()) || 0,
      type: optional(() => attachment.mimeType()) || '',
    })),
  };
}
