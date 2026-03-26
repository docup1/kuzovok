(function (window, document, shared) {
    const AVATARS = ["🐠", "🐟", "🐡", "🦈", "🐋", "🐙", "🦑", "🦀", "🐚", "🪸", "🐳", "🦐", "🦞", "🐌", "🐓", "🦆", "🦅"];

    let currentProfile = null;
    let selectedAvatar = "🐠";
    let postsCache = {};

    const elements = {
        authRequired: document.getElementById("authRequired"),
        authRequiredLoginLink: document.getElementById("authRequiredLoginLink"),
        avatarPicker: document.getElementById("avatarPicker"),
        backLink: document.getElementById("backLink"),
        editBio: document.getElementById("editBio"),
        editModal: document.getElementById("editModal"),
        editModalCloseButton: document.getElementById("editModalCloseButton"),
        editName: document.getElementById("editName"),
        logoLink: document.getElementById("logoLink"),
        postCount: document.getElementById("postCount"),
        postsContainer: document.getElementById("postsContainer"),
        profileAvatar: document.getElementById("profileAvatar"),
        profileBio: document.getElementById("profileBio"),
        profileContent: document.getElementById("profileContent"),
        profileError: document.getElementById("profileError"),
        profileName: document.getElementById("profileName"),
        profileSuccess: document.getElementById("profileSuccess"),
        profileUsername: document.getElementById("profileUsername")
    };

    function flashMessage(element, message, timeout) {
        element.textContent = message;
        element.classList.remove("hidden");
        window.setTimeout(() => element.classList.add("hidden"), timeout || 5000);
    }

    async function loadProfilePage() {
        const response = await shared.api("/profile");
        if (!response.success || !response.data) {
            flashMessage(elements.profileError, response.message || "Не удалось загрузить профиль");
            return;
        }

        currentProfile = response.data;
        elements.profileAvatar.textContent = currentProfile.avatar || "🐠";
        elements.profileName.textContent = currentProfile.name || currentProfile.username;
        elements.profileUsername.textContent = `@${currentProfile.username}`;
        elements.profileBio.textContent = currentProfile.bio || "Нет описания";
        elements.postCount.textContent = currentProfile.post_count || 0;
        document.getElementById("likeCount").textContent = currentProfile.like_count || 0;
        elements.profileContent.classList.remove("hidden");
        await loadMyPosts();
    }

    async function checkAuth() {
        const response = await shared.api("/me");
        if (response.success && response.data) {
            elements.authRequired.classList.add("hidden");
            await loadProfilePage();
            return;
        }

        elements.authRequired.classList.remove("hidden");
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

    async function loadMyPosts() {
        const response = await shared.api("/posts");
        if (response.success && Array.isArray(response.data) && response.data.length > 0) {
            postsCache = {};
            response.data.forEach((post) => {
                postsCache[post.id] = post;
            });
            elements.postsContainer.innerHTML = response.data.map(renderPost).join("");
            shared.syncImageCountdowns(elements.postsContainer);
            return;
        }

        postsCache = {};
        elements.postsContainer.innerHTML = '<div class="empty-state">У вас пока нет постов</div>';
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

    function initAvatarPicker() {
        elements.avatarPicker.innerHTML = AVATARS.map((avatar) => {
            const selectedClass = avatar === selectedAvatar ? " selected" : "";
            return `<button class="avatar-option${selectedClass}" type="button" data-avatar="${shared.esc(avatar)}">${shared.esc(avatar)}</button>`;
        }).join("");
    }

    function openEditModal() {
        selectedAvatar = elements.profileAvatar.textContent || "🐠";
        elements.editName.value = currentProfile && currentProfile.name ? currentProfile.name : "";
        elements.editBio.value = currentProfile && currentProfile.bio ? currentProfile.bio : "";
        initAvatarPicker();
        elements.editModal.classList.add("active");
    }

    function closeEditModal() {
        elements.editModal.classList.remove("active");
    }

    async function saveProfile() {
        const response = await shared.api("/profile/update", {
            method: "POST",
            body: JSON.stringify({
                _method: "PUT",
                avatar: selectedAvatar,
                name: elements.editName.value.trim(),
                bio: elements.editBio.value.trim()
            })
        });

        if (!response.success || !response.data) {
            flashMessage(elements.profileError, response.message || "Ошибка сохранения");
            return;
        }

        closeEditModal();
        flashMessage(elements.profileSuccess, "Профиль сохранён!", 3000);
        await loadProfilePage();
    }

    function bindEvents() {
        document.getElementById("editProfileButton").addEventListener("click", openEditModal);
        document.getElementById("saveProfileButton").addEventListener("click", saveProfile);
        document.getElementById("cancelEditButton").addEventListener("click", closeEditModal);
        elements.editModalCloseButton.addEventListener("click", closeEditModal);

        shared.wireModalClose(elements.editModal, closeEditModal);

        shared.delegate(elements.avatarPicker, "click", "[data-avatar]", (event, button) => {
            event.preventDefault();
            selectedAvatar = button.dataset.avatar;
            elements.avatarPicker.querySelectorAll(".avatar-option").forEach((node) => node.classList.remove("selected"));
            button.classList.add("selected");
        });

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
        elements.logoLink.href = shared.homeUrl();
        elements.backLink.href = shared.homeUrl();
        elements.authRequiredLoginLink.href = shared.homeUrl();
        bindEvents();
        checkAuth();
        window.setInterval(() => shared.syncImageCountdowns(elements.postsContainer), 1000);
    }

    init();
})(window, document, window.KuzovokShared);
