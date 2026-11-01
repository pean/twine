local terminal = "com.superlogical.terminal"
local panes = {}
local listed = rex.call("session.list")
for _, s in ipairs(listed.sessions or {}) do
  rex.call("session.attach", { session_id = s.session_id })
  local view = rex.call("session.view", { session_id = s.session_id })
  for _, w in ipairs(view.windows or {}) do
    for _, layer in ipairs(w.layers or {}) do
      for _, b in ipairs(layer.blocks or {}) do
        if b.creator_name == terminal then
          local target = { session_id = s.session_id, block_id = b.block_id, args = {} }
          local p = rex.call(terminal .. ".process", target)
          local st = rex.call(terminal .. ".program_status", target)
          local fg, child = p and p.foreground, p and p.child
          local status
          for _, r in ipairs(st and st.records or {}) do
            if status == nil or r.id == nil then status = r end
          end
          panes[#panes + 1] = {
            session = s.label,
            session_id = s.session_id,
            block = b.block_id,
            pid = (fg and fg.pid) or (child and child.pid),
            name = fg and (fg.script or fg.argv0 or fg.name),
            cwd = fg and fg.cwd,
            status = status and { state = status.state, kind = status.kind, app = status.effective_app, msg = status.msg },
          }
        end
      end
    end
  end
end
return panes
