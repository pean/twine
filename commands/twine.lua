-- Added by `twine install`. Load it from init.lua with:
--   require("twine")
--   rex.bind("ctrl+b>a", "twine_agents")

rex.action{
  name  = "twine_agents",
  title = "Twine Agents",
  run   = function()
    local _, err = rex.session.new_layer{
      bounds = { 0.05, 0.05, 0.9, 0.9 },
      focus  = true,
      layout = rex.layout.block{
        flavor  = "com.superlogical.terminal.shell",
        options = { command = { "twine", "agents" } },
      },
    }
    if err ~= nil then error(err, 0) end
  end,
}
