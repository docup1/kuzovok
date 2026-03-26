(function (window, document, shared) {
    let postsCache = {};

    const elements = {
        backLink: document.getElementById("backLink"),
        errorBackLink: document.getElementById("errorBackLink"),
        errorMessage: document.getElementById("errorMessage"),
        errorState: document.getElementById("errorState"),
        loadingState: document.getElementById("loadingState"),
        logoLink: document.getElementById("logoLink"),
        postCount: document.getElementById("postCount"),
        postsContainer: document.getElementById("postsContainer"),
        profileAvatar: document.getElementById("profileAvatar"),
        profileBio: document.getElementById("profileBio"),
        profileContent: document.getElementById("profileContent"),
        profileName: document.getElementById("profileName"),
        profileUsername: document.getElementById("profileUsername"),
        siteLogo: document.getElementById("siteLogo")
    };

    function getUsernameFromPath() {
        const match = (window.location.pathname || "").match(/\/user\/([^/]+)/);
        return match ? decodeURIComponent(match[1]) : null;
    }

    function showError(message) {
        elements.loadingState.classList.add("hidden");
        elements.errorState.classList.remove("hidden");
        elements.errorMessage.textContent = message;
    }

    function renderPost(post) {
        const likedClass = post.liked ? " liked" : "";
        const likedLabel = post.liked ? "Убрать лайк" : "Поставить лайк";
        const displayName = post.name || post.username;
        const contentHtml = post.content ? `<div class="post-content">${shared.esc(post.content)}</div>` : "";
        const imageHtml = shared.renderPostImage(post);
        const parentHtml = shared.renderParentPost(post.parent_post, { action: "open-user" });

        return `<article class="post" id="post-${post.id}">
            <div class="post-header">
                <div class="post-author">
                    <span class="post-author-avatar">${shared.esc(post.avatar || "🐠")}</span>
                    <a href="${shared.userUrl(post.username)}">${shared.esc(displayName)}</a>
                </div>
                <div class="post-date">${shared.formatDate(post.created_at)}</div>
            </div>
            ${parentHtml}
            ${contentHtml}
            ${imageHtml}
            <div class="post-footer">
                <div class="post-actions">
                    <button class="like-btn${likedClass}" type="button" aria-label="${likedLabel}" data-action="like-post" data-post-id="${post.id}">❤️ <span data-role="like-count">${post.likes}</span></button>
                </div>
            </div>
        </article>`;
    }

    async function loadUserPosts(username) {
        const response = await shared.api("/feed");
        if (!response.success || !Array.isArray(response.data)) {
            elements.postsContainer.innerHTML = '<div class="empty-state">Не удалось загрузить посты</div>';
            return;
        }

        const userPosts = response.data.filter((post) => post.username === username);
        if (!userPosts.length) {
            elements.postsContainer.innerHTML = '<div class="empty-state">У этого пользователя пока нет постов</div>';
            return;
        }

        postsCache = {};
        userPosts.forEach((post) => {
            postsCache[post.id] = post;
        });
        elements.postsContainer.innerHTML = userPosts.map(renderPost).join("");
        shared.syncImageCountdowns(elements.postsContainer);
    }

    async function loadUserProfile() {
        const username = getUsernameFromPath();
        if (!username) {
            showError("Не указан пользователь");
            return;
        }

        document.title = `${username} — Кузовок`;

        const response = await shared.api(`/users/${encodeURIComponent(username)}`);
        if (!response.success || !response.data) {
            showError(response.message || "Пользователь не найден");
            return;
        }

        const profile = response.data;
        elements.loadingState.classList.add("hidden");
        elements.profileContent.classList.remove("hidden");
        elements.profileAvatar.textContent = profile.avatar || "🐠";
        elements.profileName.textContent = profile.name || profile.username;
        elements.profileUsername.textContent = `@${profile.username}`;
        elements.profileBio.textContent = profile.bio || "Нет описания";
        elements.postCount.textContent = profile.post_count || 0;
        document.getElementById("likeCount").textContent = profile.like_count || 0;

        await loadUserPosts(username);
    }

    async function toggleLike(postId, button) {
        button.disabled = true;
        const response = await shared.api("/like", {
            method: "POST",
            body: JSON.stringify({ post_id: Number(postId) })
        });
        if (response.success) {
            shared.updateLikeButton(button, response.data);
        }
        button.disabled = false;
    }

    function bindEvents() {
        shared.delegate(elements.postsContainer, "click", "[data-action='like-post']", (event, button) => {
            event.preventDefault();
            toggleLike(button.dataset.postId, button);
        });

        shared.delegate(elements.postsContainer, "click", "[data-parent-action='open-user']", (event, node) => {
            if (event.target.closest("a")) {
                return;
            }
            window.location.href = shared.userUrl(node.dataset.parentUsername);
        });
    }

    function init() {
        shared.bootOceanScene({ bubbleCount: 10, fishCount: 3, fishSizeRange: 20, fishDurationRange: 12 });
        elements.siteLogo.src = shared.logoUrl();
        elements.logoLink.href = shared.homeUrl();
        elements.backLink.href = shared.homeUrl();
        elements.errorBackLink.href = shared.homeUrl();
        bindEvents();
        loadUserProfile();
        window.setInterval(() => shared.syncImageCountdowns(elements.postsContainer), 1000);
    }

    init();
})(window, document, window.KuzovokShared);
