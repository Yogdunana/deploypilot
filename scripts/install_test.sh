#!/bin/bash
# Tests for install.sh validation functions
# These tests define the validation functions directly to avoid executing install.sh

set -euo pipefail

# Source colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

# Override print functions to be silent during tests
print_error() { echo "ERROR: $1" >&2; }
print_warning() { echo "WARNING: $1" >&2; }
print_info() { echo "INFO: $1" >&2; }
print_success() { echo "SUCCESS: $1" >&2; }

# Copy the validation functions from install.sh
validate_port() {
    local port="$1"
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        print_error "端口必须是 1-65535 之间的数字"
        return 1
    fi
}

validate_install_dir() {
    local dir="$1"
    local bad_chars='[\$\`\|\;\&\>\<\(\)\{\}\[\]\\!~]'
    if [[ "$dir" =~ $bad_chars ]]; then
        print_error "安装路径包含非法字符"
        return 1
    fi
    if [[ "$dir" != /* ]]; then
        print_error "安装路径必须是绝对路径"
        return 1
    fi
}

validate_username() {
    local name="$1"
    if ! [[ "$name" =~ ^[a-zA-Z_][a-zA-Z0-9_-]*$ ]]; then
        print_error "用户名只能包含字母、数字、下划线和连字符，且必须以字母或下划线开头"
        return 1
    fi
    if [ ${#name} -gt 32 ]; then
        print_error "用户名长度不能超过 32 个字符"
        return 1
    fi
}

random_string() {
    local length=${1:-12}
    local result
    result=$(tr -dc 'a-z0-9' < /dev/urandom 2>/dev/null | head -c "$length") || true
    echo -n "$result"
}

generate_password() {
    local length=${1:-16}
    local result
    result=$(tr -dc 'a-zA-Z0-9!@#$%^&*' < /dev/urandom 2>/dev/null | head -c "$length") || true
    echo -n "$result"
}

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local expected_exit=0
    local actual_exit=0
    local test_name="$1"
    shift

    if "$@" >/dev/null 2>&1; then
        actual_exit=0
    else
        actual_exit=1
    fi

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo "  ✓ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "  ✗ $test_name"
        echo "    Expected exit code: $expected_exit, got: $actual_exit"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

run_test_fail() {
    local expected_exit=1
    local actual_exit=0
    local test_name="$1"
    shift

    if "$@" >/dev/null 2>&1; then
        actual_exit=0
    else
        actual_exit=1
    fi

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo "  ✓ $test_name (correctly rejected)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "  ✗ $test_name"
        echo "    Expected exit code: $expected_exit, got: $actual_exit"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# ===================== Port Validation Tests =====================
echo ""
echo "Testing validate_port():"

run_test "accepts port 1 (minimum)" validate_port 1
run_test "accepts port 80 (HTTP)" validate_port 80
run_test "accepts port 443 (HTTPS)" validate_port 443
run_test "accepts port 8080 (common)" validate_port 8080
run_test "accepts port 65535 (maximum)" validate_port 65535

run_test_fail "rejects port 0" validate_port 0
run_test_fail "rejects negative port" validate_port -1
run_test_fail "rejects port 65536" validate_port 65536
run_test_fail "rejects large port number" validate_port 100000
run_test_fail "rejects non-numeric port" validate_port abc
run_test_fail "rejects empty port" validate_port ""

# ===================== Install Directory Validation Tests =====================
echo ""
echo "Testing validate_install_dir():"

run_test "accepts /opt/deploypilot" validate_install_dir /opt/deploypilot
run_test "accepts /usr/local/bin" validate_install_dir /usr/local/bin
run_test "accepts /home/user/deploypilot" validate_install_dir /home/user/deploypilot
run_test "accepts root directory" validate_install_dir /

run_test_fail "rejects relative path" validate_install_dir opt/deploypilot
run_test_fail "rejects ./ path" validate_install_dir ./install

# Note: Space is not in the bad_chars pattern, so we skip this test
# The original script does not reject spaces in the path

# ===================== Username Validation Tests =====================
echo ""
echo "Testing validate_username():"

run_test "accepts simple username" validate_username deploy
run_test "accepts longer username" validate_username deploypilot
run_test "accepts underscores and numbers" validate_username deploy_123
run_test "accepts hyphens" validate_username deploy-user
run_test "accepts leading underscore" validate_username _deploy
run_test "accepts single character" validate_username a
run_test "accepts uppercase" validate_username AB

run_test_fail "rejects leading digit" validate_username 123deploy
run_test_fail "rejects leading hyphen" validate_username -deploy
run_test_fail "rejects @ symbol" validate_username deploy@host
run_test_fail "rejects empty username" validate_username ""

# Too long username (>32 chars)
LONG_NAME=""
for i in {1..33}; do LONG_NAME="${LONG_NAME}a"; done
run_test_fail "rejects too long username (>32 chars)" validate_username "$LONG_NAME"

# ===================== random_string Tests =====================
echo ""
echo "Testing random_string():"

# Should generate strings of correct length (the output includes a newline so wc -c will be length+1)
LEN_12=$(random_string 12 | wc -c | tr -d ' ')
if [ "$LEN_12" -eq 12 ]; then
    echo "  ✓ generates 12 character string"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "  ✗ generates 12 character string"
    echo "    Length: $LEN_12"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Should only contain lowercase alphanumeric
OUTPUT=$(random_string 50)
if [[ "$OUTPUT" =~ ^[a-z0-9]{50}$ ]]; then
    echo "  ✓ generates only lowercase alphanumeric characters"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "  ✗ generates only lowercase alphanumeric characters"
    echo "    Output: $OUTPUT"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Should generate different strings each time
STR1=$(random_string 32)
STR2=$(random_string 32)
if [ "$STR1" != "$STR2" ]; then
    echo "  ✓ generates different strings on subsequent calls"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "  ✗ generates different strings on subsequent calls"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# ===================== generate_password Tests =====================
echo ""
echo "Testing generate_password():"

# Should generate strings of correct length
LEN_PWD=$(generate_password 16 | wc -c | tr -d ' ')
if [ "$LEN_PWD" -eq 16 ]; then
    echo "  ✓ generates 16 character password"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "  ✗ generates 16 character password"
    echo "    Length: $LEN_PWD"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Should contain various character classes
PWD=$(generate_password 50)
HAS_LOWER=0
HAS_UPPER=0
HAS_DIGIT=0
HAS_SPECIAL=0

[[ "$PWD" =~ [a-z] ]] && HAS_LOWER=1
[[ "$PWD" =~ [A-Z] ]] && HAS_UPPER=1
[[ "$PWD" =~ [0-9] ]] && HAS_DIGIT=1
# Note: In bash regex, & has special meaning, so we check each special char individually
echo "$PWD" | grep -q '!' && HAS_SPECIAL=1
echo "$PWD" | grep -q '@' && HAS_SPECIAL=1
echo "$PWD" | grep -q '#' && HAS_SPECIAL=1
echo "$PWD" | grep -q '$' && HAS_SPECIAL=1
echo "$PWD" | grep -q '^' && HAS_SPECIAL=1
echo "$PWD" | grep -q '*' && HAS_SPECIAL=1

if [ $HAS_LOWER -eq 1 ] && [ $HAS_UPPER -eq 1 ] && [ $HAS_DIGIT -eq 1 ] && [ $HAS_SPECIAL -eq 1 ]; then
    echo "  ✓ generates password with all character classes"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "  ✗ generates password with all character classes"
    echo "    Has lower: $HAS_LOWER, upper: $HAS_UPPER, digit: $HAS_DIGIT, special: $HAS_SPECIAL"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# ===================== Summary =====================
echo ""
echo "========================================"
echo "Test Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
echo "========================================"

if [ $TESTS_FAILED -eq 0 ]; then
    echo "All tests passed!"
    exit 0
else
    echo "Some tests failed!"
    exit 1
fi
