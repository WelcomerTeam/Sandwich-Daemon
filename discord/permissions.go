package discord

const (
	PermissionCreateInstantInvite              = iota << 1          // Allows creation of instant invites.
	PermissionKickMembers                      = 0x0000000000000002 // Allows kicking members.
	PermissionBanMembers                       = 0x0000000000000004 // Allows banning members.
	PermissionAdministrator                    = 0x0000000000000008 // Allows all permissions and bypasses channel permission overwrites.
	PermissionManageChannels                   = 0x0000000000000010 // Allows management and editing of channels.
	PermissionManageServer                     = 0x0000000000000020 // Allows management and editing of the guild.
	PermissionAddReactions                     = 0x0000000000000040 // Allows for the addition of reactions to messages.
	PermissionViewAuditLogs                    = 0x0000000000000080 // Allows for viewing of audit logs.
	PermissionVoicePrioritySpeaker             = 0x0000000000000100 // Allows for using priority speaker in a voice channel.
	PermissionVoiceStreamVideo                 = 0x0000000000000200 // Allows the user to go live.
	PermissionViewChannel                      = 0x0000000000000400 // Allows guild members to view a channel, which includes reading messages in text channels and joining voice channels.
	PermissionSendMessages                     = 0x0000000000000800 // Allows for sending messages in a channel and creating threads in a forum (does not allow sending messages in threads).
	PermissionSendTTSMessages                  = 0x0000000000001000 // Allows for sending of /tts messages.
	PermissionManageMessages                   = 0x0000000000002000 // Allows for deletion of other users messages.
	PermissionEmbedLinks                       = 0x0000000000004000 // Links sent by users with this permission will be auto-embedded.
	PermissionAttachFiles                      = 0x0000000000008000 // Allows for uploading images and files.
	PermissionReadMessageHistory               = 0x0000000000010000 // Allows for reading of message history.
	PermissionMentionEveryone                  = 0x0000000000020000 // Allows for using the @everyone tag to notify all users in a channel, and the @here tag to notify all online users in a channel.
	PermissionUseExternalEmojis                = 0x0000000000040000 // Allows the usage of custom emojis from other servers.
	PermissionViewGuildInsights                = 0x0000000000080000 // Allows for viewing guild insights.
	PermissionVoiceConnect                     = 0x0000000000100000 // Allows for joining of a voice channel.
	PermissionVoiceSpeak                       = 0x0000000000200000 // Allows for speaking in a voice channel.
	PermissionVoiceMuteMembers                 = 0x0000000000400000 // Allows for muting members in a voice channel.
	PermissionVoiceDeafenMembers               = 0x0000000000800000 // Allows for deafening of members in a voice channel.
	PermissionVoiceMoveMembers                 = 0x0000000001000000 // Allows for moving of members between voice channels.
	PermissionVoiceUseVAD                      = 0x0000000002000000 // Allows for using voice-activity-detection in a voice channel.
	PermissionChangeNickname                   = 0x0000000004000000 // Allows for modification of own nickname.
	PermissionManageNicknames                  = 0x0000000008000000 // Allows for modification of other users nicknames.
	PermissionManageRoles                      = 0x0000000010000000 // Allows management and editing of roles.
	PermissionManageWebhooks                   = 0x0000000020000000 // Allows management and editing of webhooks.
	PermissionManageEmojis                     = 0x0000000040000000 // Allows management and editing of emojis and stickers.
	PermissionUseSlashCommands                 = 0x0000000080000000 // Allows members to use application commands, including slash commands and context menu commands.
	PermissionVoiceRequestToSpeak              = 0x0000000100000000 // Allows for requesting to speak in stage channels.
	PermissionManageEvents                     = 0x0000000200000000 // Allows for creating, editing, and deleting scheduled events.
	PermissionManageThreads                    = 0x0000000400000000 // Allows for deleting and archiving threads, and viewing all private threads.
	PermissionCreatePublicThreads              = 0x0000000800000000 // Allows for creating public and announcement threads.
	PermissionCreatePrivateThreads             = 0x0000001000000000 // Allows for creating private threads.
	PermissionUseExternalStickers              = 0x0000002000000000 // Allows the usage of custom stickers from other servers.
	PermissionSendMessagesInThreads            = 0x0000004000000000 // Allows for sending messages in threads.
	PermissionUseActivities                    = 0x0000008000000000 // Allows for using Activities (applications with the EMBEDDED flag) in a voice channel.
	PermissionModerateMembers                  = 0x0000010000000000 // Allows for timing out users to prevent them from sending or reacting to messages.
	PermissionViewCreatorMonetizationAnalytics = 0x0000020000000000 // Allows for viewing role subscription insights
	PermissionUseSoundboard                    = 0x0000040000000000 // Allows for using soundboard in a voice channel
	PermissionCreateGuildExpressions           = 0x0000080000000000 // Allows for creating emojis, stickers, and soundboard sounds, and editing and deleting those created by the current user. Not yet available to developers, see changelog.
	PermissionCreateEvents                     = 0x0000100000000000 // Allows for creating scheduled events, and editing and deleting those created by the current user. Not yet available to developers, see changelog.
	PermissionUseExternalSounds                = 0x0000200000000000 // Allows the usage of custom soundboard sounds from other servers.
	PermissionSendVoiceMessages                = 0x0000400000000000 // Allows sending voice messages.

	PermissionAllText = PermissionViewChannel |
		PermissionSendMessages |
		PermissionSendTTSMessages |
		PermissionManageMessages |
		PermissionEmbedLinks |
		PermissionAttachFiles |
		PermissionReadMessageHistory |
		PermissionMentionEveryone

	PermissionAllVoice = PermissionViewChannel |
		PermissionVoiceConnect |
		PermissionVoiceSpeak |
		PermissionVoiceMuteMembers |
		PermissionVoiceDeafenMembers |
		PermissionVoiceMoveMembers |
		PermissionVoiceUseVAD |
		PermissionVoicePrioritySpeaker

	PermissionAllChannel = PermissionAllText |
		PermissionAllVoice |
		PermissionCreateInstantInvite |
		PermissionManageRoles |
		PermissionManageChannels |
		PermissionAddReactions |
		PermissionViewAuditLogs

	PermissionAll = PermissionAllChannel |
		PermissionKickMembers |
		PermissionBanMembers |
		PermissionManageServer |
		PermissionAdministrator |
		PermissionManageWebhooks |
		PermissionManageEmojis

	PermissionElevated = PermissionKickMembers |
		PermissionBanMembers |
		PermissionAdministrator |
		PermissionManageChannels |
		PermissionManageServer |
		PermissionManageMessages |
		PermissionManageRoles |
		PermissionManageWebhooks |
		PermissionManageEmojis |
		PermissionManageThreads |
		PermissionModerateMembers
)
