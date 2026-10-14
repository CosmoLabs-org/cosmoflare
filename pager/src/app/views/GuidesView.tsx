// Guides as a real React view (FEAT-pDQEFQV + FEAT-pAGY842): the section
// lists static guidance articles as cards (title + updated date); tapping
// one navigates to "#/guides/<id>" and the view renders the article body —
// p / h3 / list blocks plus kv rows in the cf-domain-field pattern. No
// fetching — the data module is a static module, not an endpoint. The
// section masthead owns the page title (no h2 here), the same contract as
// PairingView.

import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { GUIDES, type Guide, type GuideBlock } from "../../guides";
import { guidesHref, parseGuidesHash } from "../../routes";

/** One kv block: term left, desc right — the cf-domain-field grid, the
 *  same pattern the DomainsView profile rows use. */
function KvRow({ term, desc }: { term: string; desc: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{term}</span>
      <span className="cf-domain-field-value">{desc}</span>
    </div>
  );
}

/** One body block. Static content, so the key is the block index — the
 *  array never reorders at runtime. */
function Block({ block }: { block: GuideBlock }): React.JSX.Element {
  if (block.kind === "kv") return <KvRow term={block.term} desc={block.desc} />;
  if (block.kind === "table") {
    return (
      <div className="cf-guide-table-wrap">
        <table className="cf-guide-table">
          <thead>
            <tr>{block.head.map((h, i) => <th key={i}>{h}</th>)}</tr>
          </thead>
          <tbody>
            {block.rows.map((row, r) => (
              <tr key={r}>{row.map((cell, c) => <td key={c}>{cell}</td>)}</tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  if (block.kind === "list") {
    return (
      <ul className="cf-guide-list">
        {block.text.map((item, i) => (
          <li key={i} className="cf-row-detail">{item}</li>
        ))}
      </ul>
    );
  }
  if (block.kind === "h3") return <h3 className="cf-guide-h">{block.text}</h3>;
  return <p className="cf-guide-p">{block.text}</p>;
}

/** One article: back link, title, updated date, then the body blocks. */
function GuideArticle({ guide }: { guide: Guide }): React.JSX.Element {
  return (
    <section className="cf-card cf-guide">
      <a className="cf-btn cf-btn-ghost cf-guide-back" href={guidesHref("")}>
        <span className="cf-btn-glyph" aria-hidden="true">
          <ArrowLeft size={15} strokeWidth={2.2} />
        </span>
        All guides
      </a>
      <h3 className="cf-card-title">{guide.title}</h3>
      <p className="cf-row-detail">{`Updated ${guide.updated}`}</p>
      <div className="cf-guide-body">
        {guide.body.map((block, i) => (
          <Block key={i} block={block} />
        ))}
      </div>
      {guide.source ? (
        <a className="cf-link" href={guide.source} target="_blank" rel="noreferrer">
          Source: Cloudflare docs →
        </a>
      ) : null}
    </section>
  );
}

/** The article list: one card per guide — title + updated date. */
function GuideList(): React.JSX.Element {
  return (
    <div>
      {GUIDES.map((guide) => (
        <section key={guide.id} className="cf-card">
          <h3 className="cf-card-title">
            <a href={guidesHref(guide.id)}>{guide.title}</a>
          </h3>
          <p className="cf-row-detail">{`Updated ${guide.updated}`}</p>
        </section>
      ))}
    </div>
  );
}

/** The quiet not-found card for an unknown "#/guides/<id>" target. */
function GuideNotFound({ id }: { id: string }): React.JSX.Element {
  return (
    <section className="cf-card cf-empty cf-empty-quiet">
      <h3 className="cf-card-title">No guide with that id</h3>
      <p className="cf-row-detail">
        {`There is no guide called "${id}". `}
        <a href={guidesHref("")}>Back to all guides</a>
      </p>
    </section>
  );
}

export default function GuidesView(): React.JSX.Element {
  const [articleId, setArticleId] = useState<string | null>(() => parseGuidesHash(window.location.hash));
  useEffect(() => {
    const onHash = (): void => setArticleId(parseGuidesHash(window.location.hash));
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  if (articleId !== null) {
    const guide = GUIDES.find((g) => g.id === articleId);
    if (guide) return <GuideArticle guide={guide} />;
    return <GuideNotFound id={articleId} />;
  }
  return <GuideList />;
}
