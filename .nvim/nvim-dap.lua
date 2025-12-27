local dap = require("dap")

-- precondition to using delve configuration below:
-- launch dlv from path inside project (required for dlv to find go.mod):
-- dlv dap -l 127.0.0.1:38697 --log --log-output="dap"
--
-- important! dlv needs to be launched from a path without symlinks,
-- otherwise it will report "file not found" when setting breakpoints on files
-- even when using absolute paths to them

dap.adapters.delve = {
  type = "server",
  host = "127.0.0.1",
  port = 38697,
}

dap.configurations.go = {
  {
    type = "delve",
    name = "Debug cmd",
    request = "launch",
    program = function()
      return vim.fn.getcwd() .. "/cmd/mpv-web-api"
    end,
    args = function()
      local argument_string = vim.fn.input('Program arguments: ')
      return vim.fn.split(argument_string, " ", true)
    end,
  },
  {
    type = "delve",
    name = "Debug (go.mod)",
    request = "launch",
    program = "./${relativeFileDirname}",
    args = function()
      local argument_string = vim.fn.input('Program arguments: ')
      return vim.fn.split(argument_string, " ", true)
    end,
  },
}
