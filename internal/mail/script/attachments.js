function unused(name, taken) {
  if (!taken.has(name)) return name;
  const dot = name.lastIndexOf('.');
  const stem = dot > 0 ? name.slice(0, dot) : name;
  const extension = dot > 0 ? name.slice(dot) : '';
  let copy = 2;
  while (taken.has(stem + '-' + copy + extension)) copy += 1;
  return stem + '-' + copy + extension;
}

function main(request) {
  const taken = new Set();
  return findMessage(request).message.mailAttachments().map((attachment) => {
    const name = unused(attachment.name(), taken);
    taken.add(name);
    const path = request.directory + '/' + name;
    Mail.save(attachment, { in: Path(path) });
    return { name: name, path: path };
  });
}
