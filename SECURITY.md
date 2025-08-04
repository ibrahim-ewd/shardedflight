# Security Policy

Thank you for taking the time to help make **shardedflight** safer for everyone.  
We follow a *coordinated vulnerability disclosure* model. Please read the entire document before reporting.

---

## Supported Versions

Only the latest **minor** release line receives regular security fixes.

| Version | Supported | Notes              |
|---------|-----------|--------------------|
| `v2.x`  | ✔️        | Actively patched   |
| `< v2`  | ❌        | End-of-life        |

If you cannot upgrade and need a back-port, open an Issue marked **[Security-Request]** and we will discuss options privately.

---

## Reporting a Vulnerability

1. **Email:** `git@bookshelf-writer.fun`  

2. **Include at minimum**
   - Affected version/commit hash  
   - Reproduction steps or PoC  
   - Impact assessment (confidentiality / integrity / availability)  
   - Your public PGP key for us to reply

3. We will acknowledge within **72 hours** and provide a tracking ID.

> **Please do not** open public Issues or discuss details in GitHub discussions/PRs until a fix is released.

---

## Vulnerability Handling Process

| Phase | Typical Timeframe | Description |
|-------|------------------|-------------|
| **Triage**      | ≤ 3 days   | Confirm severity, assign CVE if needed |
| **Remediation** | ≤ 14 days  | Develop & test patch, prepare advisory |
| **Pre-release** | ≤ 7 days   | Notify maintainers of downstream forks under embargo |
| **Public Release** | – | Merge fix into `main`, push tags, publish advisory |

Complex or high-risk issues may require longer; we will keep you informed.

---

## Severity Classification

We align with [CVSS 4.0](https://www.first.org/cvss/) scoring:

* **Critical (9.0–10.0):** Remote code execution, auth bypass  
* **High (7.0–8.9):** Privilege escalation, data exfiltration  
* **Medium (4.0–6.9):** Info disclosure, DoS requiring auth  
* **Low (< 4.0):** Non-exploitable or requires uncommon configuration

---

## Disclaimer

Security reports are handled on a best-effort basis by maintainers in their spare time.  
This project comes with **no warranty**; refer to the [LICENSE](./LICENSE) for details.
