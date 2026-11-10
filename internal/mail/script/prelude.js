const Mail = Application('Mail');
const preludeMissing = -1728;
const preludeInvalidIndex = -1719;
const preludeLocal = 'local';
const preludeInbox = 'inbox';
const preludeDay = 86400000;
const preludeWindows = [3, 14, 60, 365, 3650];
const preludeNotPermitted = -1743;
const preludeTimedOut = -1712;
const preludeFields = ['messageId', 'subject', 'sender', 'dateReceived', 'readStatus', 'flaggedStatus'];
// How many one-by-one reads of a property cost as much as reading it for one message
// of a whole mailbox at once (about 8 ms against 35 microseconds, measured on a
// 13,370-message mailbox). The Message-ID is 4 to 25 times dearer to read in bulk.
const preludeBulkAdvantage = 200;
const preludeMessageIdAdvantage = 25;

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

// Returns the named mailbox with its name spelled the way Mail spells it (INBOX, not
// the requested Inbox), so every command reports the same name for the same mailbox.
// A top-level match is taken as Mail lists it, without walking the whole tree.
function locateMailbox(request) {
  const owner = container(request.account);
  const requested = request.mailbox || 'Inbox';
  const wanted = requested.toLowerCase();
  const direct = preludeNames(owner.mailboxes).find((candidate) => candidate.toLowerCase() === wanted);
  if (direct !== undefined) return { mailbox: owner.mailboxes.byName(direct), name: direct };

  const entries = preludeEntries(request.account);
  const entry =
    entries.find((candidate) => candidate.name.toLowerCase() === wanted) ||
    entries.find((candidate) => candidate.leaf.toLowerCase() === wanted);
  if (entry === undefined) throw missing('mailbox', 'no mailbox named "' + requested + '" in ' + preludeWhere(request.account));
  return { mailbox: entry.mailbox, name: entry.name };
}

function findMailbox(request) {
  return locateMailbox(request).mailbox;
}

function preludeStale(error) {
  return error.errorNumber === preludeMissing || error.errorNumber === preludeInvalidIndex;
}

// A numeric id only holds inside one mailbox; a Message-ID is looked up in the named
// mailbox, or in every mailbox of the account with Inbox first. `whose(...).id()` is
// one Apple event per mailbox and yields an empty array when nothing matches.
function findMessage(request) {
  if (request.id > 0) {
    const located = locateMailbox({ account: request.account, mailbox: request.mailbox || 'Inbox' });
    const message = located.mailbox.messages.byId(request.id);
    try {
      message.id();
    } catch (error) {
      if (preludeStale(error)) throw missing('message', 'no message with id ' + request.id + ' in mailbox "' + located.name + '"');
      throw error;
    }
    return { message: message, mailbox: located.name };
  }

  const messageId = String(request.messageId || '').replace(/^<|>$/g, '');
  if (messageId === '') throw missing('message', 'no message id or Message-ID was given');

  let candidates;
  if (request.mailbox) {
    candidates = [locateMailbox(request)];
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

function preludeChanged() {
  const error = new Error('the mailbox changed while it was being read; run the command again');
  error.changed = true;
  return error;
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
  if (lengths.some((length) => length !== ids.length)) throw preludeChanged();
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

// The way of answering `matching` that has Mail do the filtering. A filtered property
// read takes seconds on a large mailbox however few rows match, so it is kept only for
// mailboxes that refuse a whole-mailbox read, and widens a date window until the window
// holds enough matches instead of reading every match.
function preludeFiltering(messages, criteria, mailboxName, limit) {
  const conditions = [];
  if (criteria.from) conditions.push({ sender: { _contains: criteria.from } });
  if (criteria.subject) conditions.push({ subject: { _contains: criteria.subject } });
  if (criteria.unread) conditions.push({ readStatus: false });
  if (criteria.since) conditions.push({ dateReceived: { _greaterThan: criteria.since } });
  if (criteria.text) conditions.push({ _or: [{ subject: { _contains: criteria.text } }, { sender: { _contains: criteria.text } }] });
  if (limit > 0 && !criteria.since) {
    for (const days of preludeWindows) {
      const since = new Date(Date.now() - days * preludeDay);
      const rows = summaries(filtered(messages, conditions.concat([{ dateReceived: { _greaterThan: since } }])), mailboxName);
      if (rows.length >= limit) return newest(rows, limit);
    }
  }
  return newest(summaries(filtered(messages, conditions), mailboxName), limit);
}

// Reads one property of every message of the mailbox, once.
function preludeColumn(table, property) {
  if (!(property in table.columns)) {
    const values = table.messages[property]();
    if (values.length !== table.ids.length) throw preludeChanged();
    table.columns[property] = values;
  }
  return table.columns[property];
}

// Reads only the properties the criteria test, for the whole mailbox, and returns the
// positions of the messages that match, newest first.
function preludeScan(messages, criteria) {
  const table = { messages: messages, ids: [], columns: {}, positions: [] };
  try {
    table.ids = messages.id() || [];
  } catch (error) {
    if (error.errorNumber === preludeMissing) return table;
    throw error;
  }
  let positions = table.ids.map((id, position) => position);
  const keep = (properties, test) => {
    if (positions.length === 0) return;
    const columns = properties.map((property) => preludeColumn(table, property));
    positions = positions.filter((position) => columns.some((values) => test(values[position])));
  };
  if (criteria.unread) keep(['readStatus'], (read) => read === false);
  if (criteria.since) keep(['dateReceived'], (date) => Boolean(date) && date > criteria.since);
  const containing = (wanted) => {
    const lowered = wanted.toLowerCase();
    return (value) => String(value || '').toLowerCase().indexOf(lowered) >= 0;
  };
  if (criteria.from) keep(['sender'], containing(criteria.from));
  if (criteria.subject) keep(['subject'], containing(criteria.subject));
  if (criteria.text) keep(['subject', 'sender'], containing(criteria.text));
  if (positions.length > 1) {
    const dates = preludeColumn(table, 'dateReceived').map((date) => preludeDate(date) || '');
    positions.sort((left, right) => dates[right].localeCompare(dates[left]));
  }
  table.positions = positions;
  return table;
}

// Builds the rows for the chosen positions. A property the scan did not read is read
// for the whole mailbox when enough rows are wanted to make that the cheaper way, and
// message by message otherwise.
function preludeRows(table, positions, mailboxName) {
  for (const property of preludeFields) {
    const advantage = property === 'messageId' ? preludeMessageIdAdvantage : preludeBulkAdvantage;
    if (positions.length * advantage >= table.ids.length) preludeColumn(table, property);
  }
  return positions.map((position) => {
    const id = table.ids[position];
    const message = table.messages.byId(id);
    const value = (property) => (property in table.columns ? table.columns[property][position] : message[property]());
    try {
      return {
        id: id,
        messageId: value('messageId') || '',
        mailbox: mailboxName || '',
        subject: value('subject') || '',
        sender: value('sender') || '',
        received: preludeDate(value('dateReceived')),
        read: value('readStatus'),
        flagged: value('flaggedStatus'),
      };
    } catch (error) {
      if (preludeStale(error)) throw preludeChanged();
      throw error;
    }
  });
}

// Returns the newest `limit` messages (all of them for 0) that match the criteria:
// unread, since (a Date), and from, subject and text, which match without regard to
// case. Reading a property of every message of a mailbox is far cheaper than having
// Mail filter first, so the filtering and sorting happen here and the remaining
// properties are read only for the rows returned.
function matching(messages, criteria, mailboxName, limit) {
  try {
    const table = preludeScan(messages, criteria);
    const positions = limit > 0 ? table.positions.slice(0, limit) : table.positions;
    return preludeRows(table, positions, mailboxName);
  } catch (error) {
    if (error.changed || error.errorNumber === preludeNotPermitted || error.errorNumber === preludeTimedOut) throw error;
    return preludeFiltering(messages, criteria, mailboxName, limit);
  }
}
