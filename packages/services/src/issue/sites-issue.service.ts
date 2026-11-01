/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { API_BASE_URL } from "@plane/constants";
import type { IPublicIssue, TIssuePublicComment, TPublicIssuesResponse } from "@plane/types";
// api service
import { APIService } from "../api.service";

/**
 * Service class for managing issues within plane sites application
 * Extends the APIService class to handle HTTP requests to the issue-related endpoints
 * @extends {APIService}
 * @remarks This service is only available for plane sites
 */
export class SitesIssueService extends APIService {
  constructor(BASE_URL?: string) {
    super(BASE_URL || API_BASE_URL);
  }

  /**
   * Retrieves a paginated list of issues for a specific anchor
   * @param {string} anchor - The anchor identifier
   * @param {object} params - Optional query parameters
   * @returns {Promise<TPublicIssuesResponse>} Promise resolving to a paginated list of issues
   * @throws {Error} If the API request fails
   */
  async list(
    anchor: string,
    params: Record<string, string | number | boolean | null | undefined>,
    signal?: AbortSignal
  ): Promise<TPublicIssuesResponse> {
    return this.responseData(
      this.get(`/api/public/anchor/${anchor}/issues/`, {
        params,
        signal,
      }),
      "response-or-error"
    );
  }

  /**
   * Retrieves details of a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @returns {Promise<IPublicIssue>} Promise resolving to the issue details
   * @throws {Error} If the API request fails
   */
  async retrieve(anchor: string, issueID: string, signal?: AbortSignal): Promise<IPublicIssue> {
    return this.responseData(
      this.get(`/api/public/anchor/${anchor}/issues/${issueID}/`, { signal }),
      "response-or-error"
    );
  }

  /**
   * Retrieves the votes associated with a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @returns {Promise<unknown>} Promise resolving to the votes
   * @throws {Error} If the API request fails
   */
  async listVotes(anchor: string, issueID: string): Promise<unknown> {
    return this.responseData(this.get(`/api/public/anchor/${anchor}/issues/${issueID}/votes/`), "response-or-error");
  }

  /**
   * Creates a new vote for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {object} data - The vote data
   * @returns {Promise<unknown>} Promise resolving to the created vote
   * @throws {Error} If the API request fails
   */
  async addVote(anchor: string, issueID: string, data: { vote: -1 | 1 }): Promise<unknown> {
    return this.responseData(
      this.post(`/api/public/anchor/${anchor}/issues/${issueID}/votes/`, data),
      "response-or-error"
    );
  }

  /**
   * Deletes a vote for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @returns {Promise<unknown>} Promise resolving to the deletion response
   * @throws {Error} If the API request fails
   */
  async removeVote(anchor: string, issueID: string): Promise<void> {
    return this.delete(`/api/public/anchor/${anchor}/issues/${issueID}/votes/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  /**
   * Retrieves the reactions associated with a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @returns {Promise<unknown>} Promise resolving to the reactions
   * @throws {Error} If the API request fails
   */
  async listReactions(anchor: string, issueID: string): Promise<unknown> {
    return this.responseData(
      this.get(`/api/public/anchor/${anchor}/issues/${issueID}/reactions/`),
      "response-or-error"
    );
  }

  /**
   * Creates a new reaction for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {object} data - The reaction data
   * @returns {Promise<unknown>} Promise resolving to the created reaction
   * @throws {Error} If the API request fails
   */
  async addReaction(anchor: string, issueID: string, data: { reaction: string }): Promise<unknown> {
    return this.responseData(
      this.post(`/api/public/anchor/${anchor}/issues/${issueID}/reactions/`, data),
      "response-or-error"
    );
  }

  /**
   * Deletes a reaction for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {string} reactionId - The reaction identifier
   * @returns {Promise<unknown>} Promise resolving to the deletion response
   * @throws {Error} If the API request fails
   */
  async removeReaction(anchor: string, issueID: string, reactionId: string): Promise<void> {
    return this.delete(`/api/public/anchor/${anchor}/issues/${issueID}/reactions/${reactionId}/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  /**
   * Retrieves the comments associated with a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @returns {Promise<unknown>} Promise resolving to the comments
   * @throws {Error} If the API request fails
   */
  async listComments(anchor: string, issueID: string, signal?: AbortSignal): Promise<TIssuePublicComment[]> {
    return this.responseData(
      this.get(`/api/public/anchor/${anchor}/issues/${issueID}/comments/`, { signal }),
      "response-or-error"
    );
  }

  /**
   * Creates a new comment for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {object} data - The comment data
   * @returns {Promise<TIssuePublicComment>} Promise resolving to the created comment
   * @throws {Error} If the API request fails
   */
  async addComment(anchor: string, issueID: string, data: Partial<TIssuePublicComment>): Promise<TIssuePublicComment> {
    return this.responseData(
      this.post(`/api/public/anchor/${anchor}/issues/${issueID}/comments/`, data),
      "response-or-error"
    );
  }

  /**
   * Updates a comment for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {string} commentId - The comment identifier
   * @param {object} data - The updated comment data
   * @returns {Promise<unknown>} Promise resolving to the updated comment
   * @throws {Error} If the API request fails
   */
  async updateComment(
    anchor: string,
    issueID: string,
    commentId: string,
    data: Partial<TIssuePublicComment>
  ): Promise<TIssuePublicComment> {
    return this.responseData(
      this.patch(`/api/public/anchor/${anchor}/issues/${issueID}/comments/${commentId}/`, data),
      "response-or-error"
    );
  }

  /**
   * Deletes a comment for a specific issue
   * @param {string} anchor - The anchor identifier
   * @param {string} issueID - The issue identifier
   * @param {string} commentId - The comment identifier
   * @returns {Promise<unknown>} Promise resolving to the deletion response
   * @throws {Error} If the API request fails
   */
  async removeComment(anchor: string, issueID: string, commentId: string): Promise<void> {
    return this.delete(`/api/public/anchor/${anchor}/issues/${issueID}/comments/${commentId}/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  /**
   * Creates a new reaction for a specific comment
   * @param {string} anchor - The anchor identifier
   * @param {string} commentId - The comment identifier
   * @param {object} data - The reaction data
   * @returns {Promise<unknown>} Promise resolving to the created reaction
   * @throws {Error} If the API request fails
   */
  async addCommentReaction(
    anchor: string,
    commentId: string,
    data: {
      reaction: string;
    }
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/public/anchor/${anchor}/comments/${commentId}/reactions/`, data),
      "response-or-error"
    );
  }

  /**
   * Deletes a reaction for a specific comment
   * @param {string} anchor - The anchor identifier
   * @param {string} commentId - The comment identifier
   * @param {string} reactionHex - The reaction identifier
   * @returns {Promise<unknown>} Promise resolving to the deletion response
   * @throws {Error} If the API request fails
   */
  async removeCommentReaction(anchor: string, commentId: string, reactionHex: string): Promise<void> {
    return this.delete(`/api/public/anchor/${anchor}/comments/${commentId}/reactions/${reactionHex}/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }
}
