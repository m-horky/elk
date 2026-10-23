# Security policy

This is an upstream project and is not affiliated with any Red Hat product.

## Reporting a vulnerability

Report security vulnerabilities privately through [GitHub Security Advisories](https://github.com/m-horky/elk/security/advisories/new). Include the affected commit, the impact, and reproducible steps.

## Disclaimer

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR THIS SOFTWARE.

---

# Threat model

## Security-relevant behavior

The following points are known and expected features, and should not be considered security findings.

- TLS verification and HTTP-only connections can be set in configuration. They are meant to be used for development and on trusted networks. All cleartext leakage through HTTP is out of scope.
- Filesystem reads can follow symlinks. All paths are assumed trusted since they are not attacker-controllable. Symlinks are a feature of a filesystem.
- Paths read by the program can be configured with environment variables, which are not attacker-controllable. Customizable paths are only supported for development and testing. This includes the locations of the external tools the program executes for fact collection (`virt-what`, `lscpu`), so the process environment must never be influenced by untrusted input; the variables are resolved before any privilege boundary is crossed by the caller, never by this program itself.
- The fact collectors read root-owned system resources (`os-release`, `uname(2)`) and run root-owned system tools (`virt-what`, `lscpu`). The resulting fact values are published to the remote API as-is and are not sanitized, because every source is assumed trusted.
- Consumer identity, entitlement, and product certificate locations under `/etc/pki/` are system paths shared with subscription-manager; their permissions and ownership are managed by the packaging, not by this program.
