import sys

stack = []
with open('final.stacks', 'r') as f:
    for line in f:
        line = line.strip()
        # Skip garbage lines
        if not line or line.startswith('Attaching'):
            continue
            
        if line.startswith('@stacks['):
            stack = []
        elif line.startswith(']:'):
            # End of stack. Format: "]: <count>"
            try:
                count = line.split(':')[1].strip()
                # FlameGraphs need Root -> Leaf, but bpftrace gives Leaf -> Root.
                # So we reverse the stack.
                folded_stack = ";".join(reversed(stack))
                print(f"{folded_stack} {count}")
            except:
                pass
        else:
            # This is a stack frame. 
            # We strip off the "+1234" offsets to make the graph cleaner.
            func_name = line.split('+')[0].strip()
            stack.append(func_name)
