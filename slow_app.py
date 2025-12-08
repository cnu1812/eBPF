from flask import Flask
import re
import time

app = Flask(__name__)

# A deliberately inefficient regex
def very_slow_regex(input_str):
    # This loop + regex combo is designed to be CPU-intensive
    for _ in range(5000):
        re.search(r'(a|b|c|d|e|f|g)+z', input_str)

@app.route('/')
def fast_route():
    return "This is a fast route!\n"

@app.route('/slow')
def slow_route():
    start_time = time.time()
    very_slow_regex("a" * 100) # Run the slow function
    duration = time.time() - start_time
    return f"This was a slow route! Duration: {duration:.2f}s\n"

if __name__ == '__main__':
    # Important: 'debug=False' ensures it runs in a single, profile-able process
    app.run(debug=False, host='127.0.0.1', port=5000)
