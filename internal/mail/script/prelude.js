const Mail = Application('Mail');
const preludeMissing = -1728;
const preludeInvalidIndex = -1719;
const preludeLocal = 'local';
const preludeInbox = 'inbox';

function run(argv) {
  try {
    const request = JSON.parse((argv && argv[0]) || '{}');
    const result = main(request);
    return JSON.stringify({ result: result === undefined ? null : result });
  } catch (error) {
    return JSON.stringify({
      error: {
        number: (error && error.errorNumber) || 0,
        message: String((error && error.message) || error),
        kind: (error && error.kind) || '',
      },
    });
  }
}

function missing(kind, text) {
  const error = new Error(text);
  error.kind = kind;
  error.errorNumber = preludeMissing;
  return error;
}

function refuse(text) {
  const error = new Error(text);
  error.kind = 'refused';
  return error;
}

function optional(getter) {
  try {
    return getter();
  } catch (error) {
    return '';
  }
}

function addresses(recipients) {
  let names;
  let values;
  try {
    names = recipients.name();
    values = recipients.address();
  } catch (error) {
    if (error.errorNumber === preludeMissing) return [];
    throw error;
  }
  return values.map((address, position) => ({ name: names[position] || '', address: address || '' }));
}

function findAccount(name) {
  const account = Mail.accounts.byName(name);
  if (!account.exists()) throw missing('account', 'no account named "' + name + '" in Mail');
  return account;
}

function container(account) {
  return account === preludeLocal ? Mail : findAccount(account);
}

function preludeNames(collection) {
  try {
    return collection.name() || [];
  } catch (error) {
    if (error.errorNumber === preludeMissing) return [];
    throw error;
  }
}

function preludeLeaf(name) {
  return name.slice(name.lastIndexOf('/') + 1);
}

// Returns every mailbox of an account (or of Mail itself for local) with its full
// `Parent/Child` path. Mail's top-level mailbox list also repeats nested mailboxes
// under their leaf name (or full path), so a top-level entry whose name turns up
// nested elsewhere is a duplicate and is dropped with everything below it. A genuine
// top-level mailbox sharing its name with a nested one is indistinguishable here.
function preludeEntries(account) {
  const entries = [];
  function visit(collection, parent, root, depth) {
    preludeNames(collection).forEach((name, position) => {
      const mailbox = collection[position];
      const path = parent === '' || name.indexOf('/') >= 0 ? name : parent + '/' + name;
      const top = root === null ? name : root;
      entries.push({ mailbox: mailbox, name: path, leaf: preludeLeaf(path), root: top, depth: depth });
      visit(mailbox.mailboxes, path, top, depth + 1);
    });
  }
  visit(container(account).mailboxes, '', null, 0);

  const nested = new Set();
  for (const entry of entries) {
    if (entry.depth > 0) {
      nested.add(entry.leaf.toLowerCase());
      nested.add(entry.name.toLowerCase());
    }
  }
  const seen = new Set();
  return entries.filter((entry) => {
    const root = entry.root.toLowerCase();
    if (root.indexOf('/') >= 0 || nested.has(root)) return false;
    const key = entry.name.toLowerCase();
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function mailboxesOf(account) {
  return preludeEntries(account).map((entry) => entry.mailbox);
}

function preludeWhere(account) {
  return account === preludeLocal ? 'local mailboxes (On My Mac)' : 'account "' + account + '"';
}

function findMailbox(request) {
  const owner = container(request.account);
  const requested = request.mailbox || 'Inbox';
  const wanted = requested.toLowerCase();
  const direct = preludeNames(owner.mailboxes).find((candidate) => candidate.toLowerCase() === wanted);
  if (direct !== undefined) return owner.mailboxes.byName(direct);

  const entries = preludeEntries(request.account);
  const entry =
    entries.find((candidate) => candidate.name.toLowerCase() === wanted) ||
    entries.find((candidate) => candidate.leaf.toLowerCase() === wanted);
  if (entry === undefined) throw missing('mailbox', 'no mailbox named "' + requested + '" in ' + preludeWhere(request.account));
  return entry.mailbox;
}

function preludeStale(error) {
  return error.errorNumber === preludeMissing || error.errorNumber === preludeInvalidIndex;
}

// A numeric id only holds inside one mailbox; a Message-ID is looked up in the named
// mailbox, or in every mailbox of the account with Inbox first. `whose(...).id()` is
// one Apple event per mailbox and yields an empty array when nothing matches.
function findMessage(request) {
  if (request.id > 0) {
    const mailboxName = request.mailbox || 'Inbox';
    const message = findMailbox({ account: request.account, mailbox: mailboxName }).messages.byId(request.id);
    try {
      message.id();
    } catch (error) {
      if (preludeStale(error)) throw missing('message', 'no message with id ' + request.id + ' in mailbox "' + mailboxName + '"');
      throw error;
    }
    return { message: message, mailbox: mailboxName };
  }

  const messageId = String(request.messageId || '').replace(/^<|>$/g, '');
  if (messageId === '') throw missing('message', 'no message id or Message-ID was given');

  let candidates;
  if (request.mailbox) {
    candidates = [{ mailbox: findMailbox(request), name: request.mailbox }];
  } else {
    const entries = preludeEntries(request.account);
    candidates = entries
      .filter((entry) => entry.leaf.toLowerCase() === preludeInbox)
      .concat(entries.filter((entry) => entry.leaf.toLowerCase() !== preludeInbox));
  }
  for (const candidate of candidates) {
    let ids;
    try {
      ids = candidate.mailbox.messages.whose({ messageId: messageId }).id();
    } catch (error) {
      if (preludeStale(error)) continue;
      throw error;
    }
    if (ids && ids.length > 0) return { message: candidate.mailbox.messages.byId(ids[0]), mailbox: candidate.name };
  }
  const scope = request.mailbox ? 'mailbox "' + request.mailbox + '" of ' + preludeWhere(request.account) : preludeWhere(request.account);
  throw missing('message', 'no message with Message-ID <' + messageId + '> in ' + scope);
}

function preludeDate(value) {
  return value ? value.toISOString() : null;
}

function describe(message, mailboxName) {
  return {
    id: message.id(),
    messageId: optional(() => message.messageId()) || '',
    mailbox: mailboxName || '',
    subject: optional(() => message.subject()) || '',
    sender: optional(() => message.sender()) || '',
    received: preludeDate(optional(() => message.dateReceived())),
    read: message.readStatus(),
    flagged: message.flaggedStatus(),
  };
}

// Asking Mail for one property of many messages is a single Apple event, while
// walking messages one by one costs several events each.
function summaries(specifier, mailboxName) {
  let ids;
  try {
    ids = specifier.id();
  } catch (error) {
    if (error.errorNumber === preludeMissing) return [];
    throw error;
  }
  const messageIds = specifier.messageId();
  const subjects = specifier.subject();
  const senders = specifier.sender();
  const dates = specifier.dateReceived();
  const reads = specifier.readStatus();
  const flags = specifier.flaggedStatus();
  const lengths = [messageIds, subjects, senders, dates, reads, flags].map((values) => values.length);
  if (lengths.some((length) => length !== ids.length)) throw new Error('the mailbox changed while it was being read; run the command again');
  return ids.map((id, position) => ({
    id: id,
    messageId: messageIds[position] || '',
    mailbox: mailboxName || '',
    subject: subjects[position] || '',
    sender: senders[position] || '',
    received: preludeDate(dates[position]),
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
