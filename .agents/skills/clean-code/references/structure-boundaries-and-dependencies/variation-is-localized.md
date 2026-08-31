# Variation is localized

When behavior varies, the points of variation are narrow and explicit rather than smeared through conditionals everywhere.

Variation should have an intentional home, such as a strategy, configuration seam, or small decision point. When variation is spread across many branches, the code becomes difficult to extend safely. In review, this is about whether the solution has more moving parts than the requirement has earned. Strong signals are a small number of concepts, one obvious route through the logic, and abstractions that remove repeated cost instead of adding ceremony. Weak signals are speculative generalization, many special cases, flag-driven behavior, and repeated domain rules hiding inside primitive data. The educational point is that unnecessary complexity compounds maintenance cost and makes every later bug harder to isolate. For this specific symptom, the reviewer should ask whether the change makes 'Variation is localized' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
Checkout.sendConfirmation(order, channel):
  if channel == "email": Email.sendReceipt(order)
  else: Sms.sendReceipt(order)
Returns.sendConfirmation(refund, channel):
  if channel == "email": Email.sendRefund(refund)
  else: Sms.sendRefund(refund)
```

### Good

```text
NotificationChannel.for(name):
  if name == "email": return Email
  else: return Sms
Checkout.sendConfirmation(order, channelName):
  channel = NotificationChannel.for(channelName)
  channel.sendReceipt(order)
Returns.sendConfirmation(refund, channelName):
  channel = NotificationChannel.for(channelName)
  channel.sendRefund(refund)
```
