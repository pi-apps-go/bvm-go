author = "Botspot"
-- based on rdp-ntlm-info.nse script with help from chatgpt


license = "Same as Nmap--See https://nmap.org/book/man-legal.html"
categories = {"default", "discovery", "safe"}

-- changed by botspot to match any port number
portrule = function(host, port)
    return port.protocol == "tcp"
end

local rdp = require "rdp"

action = function(host, port)
  local comm = rdp.Comm:new(host, port)
  if not comm:connect() then
    return nil
  end

  local requested_protocol = rdp.PROTOCOL_SSL | rdp.PROTOCOL_HYBRID | rdp.PROTOCOL_HYBRID_EX
  local cr = rdp.Request.ConnectionRequest:new(requested_protocol)
  local status, _ = comm:exch(cr)

  comm:close()

  if status then
    return "RDP service detected"
  end

  return nil
end
