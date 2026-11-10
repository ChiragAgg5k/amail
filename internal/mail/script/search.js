function run(argv) {
  const request = JSON.parse(argv[0]);
  const conditions = [];
  if (request.text) conditions.push({ _or: [{ subject: { _contains: request.text } }, { sender: { _contains: request.text } }] });
  if (request.from) conditions.push({ sender: { _contains: request.from } });
  if (request.subject) conditions.push({ subject: { _contains: request.subject } });
  if (request.unread) conditions.push({ readStatus: false });
  if (request.since) conditions.push({ dateReceived: { _greaterThan: new Date(request.since) } });
  const rows = summaries(filtered(findMailbox(request).messages, conditions));
  return JSON.stringify(newest(rows, request.limit));
}
