local function run_tests()
   local failed = 0
   local total = 0
   
   for file in io.popen('ls *.ds'):lines() do
       total = total + 1
       
       -- compile test
       local build = os.execute("dash build " .. file)
       if not build then
           print("Build failed: " .. file)
           failed = failed + 1
           goto continue
       end

       -- run test
       local base = file:gsub("%.ds$", "")
       local exec = os.execute("./" .. base)
       if not exec then
           print("Execution failed: " .. base)
           failed = failed + 1
           goto continue
       end

       os.remove(base)
       
       ::continue::
   end

   if failed == 0 then
       print(string.format("%d/%d tests succeeded", total, total))
   else
       print(string.format("%d/%d tests failed", failed, total))
       os.exit(1)
   end
end

run_tests()
