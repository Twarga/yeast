---
hide:
  - toc
---

<div class="academy-shell">
  <div class="academy-shell__inner">

    <a class="academy-back" href="../index.md">Academy home</a>

    <div class="academy-page-hero">
      <div class="academy-section-label">Twarga Academy · Source Contract</div>
      <h1>The rules that keep the course honest.</h1>
      <p>
        Twarga Academy mirrors the external bootcamp source. These documents are the constraints
        the course operates within — they prevent scope creep and keep every lab grounded in what
        Yeast v1.1 actually does.
      </p>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">Core rules</div>
      <ul class="academy-rule-list">
        <li>Use only Yeast v1.1 features documented in the supported surface.</li>
        <li>Browser access from the laptop must use SSH local port forwarding. Yeast does not expose guest ports directly.</li>
        <li><code>yeast ssh</code> is interactive and supports <code>--verbose</code>. It is not a raw SSH argument passthrough.</li>
        <li>Yeast v1.1 does not have a general <code>ports:</code> field in <code>yeast.yaml</code>. Use <code>ssh_port</code> only.</li>
        <li>If the source files and the Yeast repo ever disagree, the source files win.</li>
        <li>Treat the mirrored lab folders as the public academy content. Do not rewrite them without updating the source.</li>
      </ul>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">Source documents</div>
      <div class="academy-link-grid">
        <a href="../CURRICULUM.md">
          <strong>CURRICULUM.md</strong><br>
          <span style="color:var(--yeast-text-secondary);font-weight:400;font-size:0.9rem">The authoritative 30-lab course outline and lab structure rules.</span>
        </a>
        <a href="../ACCESS.md">
          <strong>ACCESS.md</strong><br>
          <span style="color:var(--yeast-text-secondary);font-weight:400;font-size:0.9rem">Browser and service exposure rules — how to reach guest services from the laptop.</span>
        </a>
        <a href="../SUPPORTED_YEAST_V1_1.md">
          <strong>SUPPORTED_YEAST_V1_1.md</strong><br>
          <span style="color:var(--yeast-text-secondary);font-weight:400;font-size:0.9rem">The exact Yeast v1.1 surface the course is allowed to use.</span>
        </a>
        <a href="../HARDENING_PLAN.md">
          <strong>HARDENING_PLAN.md</strong><br>
          <span style="color:var(--yeast-text-secondary);font-weight:400;font-size:0.9rem">Known gaps and the plan for closing them before public launch.</span>
        </a>
        <a href="../PROGRESS.md">
          <strong>PROGRESS.md</strong><br>
          <span style="color:var(--yeast-text-secondary);font-weight:400;font-size:0.9rem">Lab completion status and current progress through the 30-lab set.</span>
        </a>
      </div>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">What this means in practice</div>
      <div class="academy-grid academy-grid--three">
        <div class="academy-card academy-card--soft">
          <h3 style="font-size:1rem;letter-spacing:-0.01em">Labs stay inside v1.1</h3>
          <p style="font-size:0.9rem">No lab invents Yeast features. If a lab needs something Yeast can't do, it works around it explicitly.</p>
        </div>
        <div class="academy-card academy-card--soft">
          <h3 style="font-size:1rem;letter-spacing:-0.01em">Browser access via SSH tunnel</h3>
          <p style="font-size:0.9rem">Every lab that serves a UI explains SSH local port forwarding before asking you to open a browser.</p>
        </div>
        <div class="academy-card academy-card--soft">
          <h3 style="font-size:1rem;letter-spacing:-0.01em">Source is authoritative</h3>
          <p style="font-size:0.9rem">The mirrored lab folders are the course. The web presentation is secondary — the source wins on conflicts.</p>
        </div>
      </div>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">Continue the course</div>
      <div class="academy-link-grid academy-link-grid--emphasis">
        <a href="../curriculum/index.md"><strong>Curriculum</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Full 30-lab map with phase overview.</span></a>
        <a href="../modules/index.md"><strong>Modules</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Browse the course in five themed blocks.</span></a>
        <a href="../labs/index.md"><strong>Labs</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Open the hands-on lab set.</span></a>
      </div>
    </div>

  </div>
</div>
