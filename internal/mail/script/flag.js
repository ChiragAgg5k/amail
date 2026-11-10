// Mail's dictionary documents flag index only as -1 for none; the colour order
// (0 red, 1 orange, 2 yellow, 3 green, 4 blue, 5 purple, 6 gray) follows Mail's
// Flag menu as reported by the scripting community and needs confirming in the e2e suite.
// Ticket 7 assumption, verify when built: flagging keeps the numeric id.
//
// The dictionary maps `flag index` to the Cocoa key flaggedStatus and `flagged
// status` to isFlagged, so writing flagged status after the index can reset the
// colour. A colour is therefore written last, after marking the message flagged.
function main(request) {
  const found = findMessage(request);
  const index = Number(request.index);
  if (!Number.isInteger(index)) throw new Error('flag index must be an integer, got ' + request.index);
  if (index < 0) {
    found.message.flaggedStatus = false;
  } else {
    found.message.flaggedStatus = true;
    found.message.flagIndex = index;
  }
  return describe(found.message, found.mailbox);
}
