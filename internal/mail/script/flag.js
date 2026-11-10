// Mail's dictionary documents flag index only as -1 for none; the colour order
// (0 red, 1 orange, 2 yellow, 3 green, 4 blue, 5 purple, 6 gray) follows Mail's
// Flag menu as reported by the scripting community and needs confirming in the e2e suite.
// Ticket 7 assumption, verify when built: flagging keeps the numeric id.
function main(request) {
  const found = findMessage(request);
  const index = Number(request.index);
  if (!Number.isInteger(index)) throw new Error('flag index must be an integer, got ' + request.index);
  if (index < 0) {
    found.message.flaggedStatus = false;
  } else {
    found.message.flagIndex = index;
    found.message.flaggedStatus = true;
  }
  return describe(found.message, found.mailbox);
}
