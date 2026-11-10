const Mail = Application('Mail');
const missingObject = -1728;

function findAccount(name) {
  const account = Mail.accounts.byName(name);
  if (!account.exists()) throw new Error('no account named "' + name + '" in Mail');
  return account;
}

function findMailbox(request) {
  const account = findAccount(request.account);
  const wanted = request.mailbox.toLowerCase();
  const name = account.mailboxes.name().find((candidate) => candidate.toLowerCase() === wanted);
  if (name === undefined) throw new Error('no mailbox named "' + request.mailbox + '" in account "' + request.account + '"');
  return account.mailboxes.byName(name);
}

function findMessage(request) {
  const message = findMailbox(request).messages.byId(request.id);
  if (!message.exists()) throw new Error('no message with id ' + request.id + ' in mailbox "' + request.mailbox + '"');
  return message;
}

function summaries(specifier) {
  let ids;
  try {
    ids = specifier.id();
  } catch (error) {
    if (error.errorNumber === missingObject) return [];
    throw error;
  }
  const subjects = specifier.subject();
  const senders = specifier.sender();
  const dates = specifier.dateReceived();
  const reads = specifier.readStatus();
  const flags = specifier.flaggedStatus();
  const lengths = [subjects, senders, dates, reads, flags].map((values) => values.length);
  if (lengths.some((length) => length !== ids.length)) throw new Error('the mailbox changed while it was being read; run the command again');
  return ids.map((id, position) => ({
    id: id,
    subject: subjects[position] || '',
    sender: senders[position] || '',
    received: dates[position] ? dates[position].toISOString() : null,
    read: reads[position],
    flagged: flags[position],
  }));
}

function newest(rows, limit) {
  rows.sort((left, right) => (right.received || '').localeCompare(left.received || ''));
  return limit > 0 ? rows.slice(0, limit) : rows;
}

function filtered(messages, conditions) {
  if (conditions.length === 0) return messages;
  if (conditions.length === 1) return messages.whose(conditions[0]);
  return messages.whose({ _and: conditions });
}
